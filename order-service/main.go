package main

import (
	"log"
	"net/http"
	"order-service/controllers"
	"order-service/database"
	"order-service/middlewares"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseDB()

	// 创建Gin路由
	r := gin.Default()

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 需要认证的路由组
	authGroup := r.Group("/api")
	authGroup.Use(middlewares.AuthMiddleware())
	{
		authGroup.POST("/orders", controllers.CreateOrder)
		authGroup.GET("/orders", controllers.GetUserOrders)
		authGroup.GET("/orders/:id", controllers.GetOrderDetails)
		authGroup.PUT("/orders/:id/status", controllers.UpdateOrderStatus)
	}

	// 启动服务器
	port := ":8080"
	log.Printf("Order service starting on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
