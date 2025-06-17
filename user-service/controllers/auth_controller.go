package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
	"user-service/database"
	"user-service/models"
	"user-service/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO users (username, email, password, created_at) VALUES (?, ?, ?, ?)",
		user.Username,
		user.Email,
		string(hashedPassword),
		time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create user"})
		return
	}

	userID, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": userID, "message": "User created"})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		"SELECT id, password FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Password)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func ProtectedEndpoint(c *gin.Context) {
	userID, _ := c.Get("userID")
	c.JSON(http.StatusOK, gin.H{"message": "Access granted", "user_id": userID})
}
