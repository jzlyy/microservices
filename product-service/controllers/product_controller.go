package controllers

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"product-service/database"
	"product-service/models"
	"product-service/utils"
	"strconv"
)

func CreateCategory(c *gin.Context) {
	var category models.Category
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := database.DB.Exec(
		"INSERT INTO categories (name, description) VALUES (?, ?)",
		category.Name, category.Description,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category"})
		return
	}

	categoryID, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": categoryID})
}

func CreateProduct(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证分类是否存在
	var exists bool
	err := database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)",
		product.CategoryID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// 开始事务
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start transaction"})
		return
	}

	// 插入产品
	result, err := tx.Exec(
		`INSERT INTO products 
		(name, description, price, stock, category_id, sku, image_url) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		product.Name, product.Description, product.Price, product.Stock,
		product.CategoryID, product.SKU, product.ImageURL,
	)
	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product"})
		return
	}

	productID, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": productID})
}

func GetProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// 查询产品基本信息
	var product models.ProductDetail
	err = database.DB.QueryRow(`
		SELECT p.id, p.name, p.description, p.price, p.stock, p.category_id, 
		       p.sku, p.image_url, p.created_at, p.updated_at, c.name AS category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id
		WHERE p.id = ? AND p.deleted_at IS NULL
	`, productID).Scan(
		&product.ID, &product.Name, &product.Description, &product.Price,
		&product.Stock, &product.CategoryID, &product.SKU, &product.ImageURL,
		&product.CreatedAt, &product.UpdatedAt, &product.CategoryName,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// 查询产品属性
	rows, err := database.DB.Query(`
		SELECT id, name, value
		FROM product_attributes
		WHERE product_id = ?
	`, productID)
	if err != nil {
		log.Printf("Error fetching attributes: %v", err)
	} else {
		defer func(rows *sql.Rows) {
			err := rows.Close()
			if err != nil {

			}
		}(rows)
		for rows.Next() {
			var attr models.ProductAttribute
			if err := rows.Scan(&attr.ID, &attr.Name, &attr.Value); err != nil {
				log.Printf("Error scanning attribute: %v", err)
				continue
			}
			product.Attributes = append(product.Attributes, attr)
		}
	}

	// 查询产品图片
	imgRows, err := database.DB.Query(`
		SELECT id, image_url, is_primary
		FROM product_images
		WHERE product_id = ?
	`, productID)
	if err != nil {
		log.Printf("Error fetching images: %v", err)
	} else {
		defer func(imgRows *sql.Rows) {
			err := imgRows.Close()
			if err != nil {

			}
		}(imgRows)
		for imgRows.Next() {
			var img models.ProductImage
			if err := imgRows.Scan(&img.ID, &img.ImageURL, &img.IsPrimary); err != nil {
				log.Printf("Error scanning image: %v", err)
				continue
			}
			product.Images = append(product.Images, img)
		}
	}

	c.JSON(http.StatusOK, product)
}

func ListProducts(c *gin.Context) {
	var filter models.ProductFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pagination := utils.ParsePagination(c)

	// 构建查询条件
	query := `
		SELECT p.id, p.name, p.description, p.price, p.stock, p.category_id, 
		       p.sku, p.image_url, p.created_at, p.updated_at, c.name AS category_name
		FROM products p
		JOIN categories c ON p.category_id = c.id
		WHERE p.deleted_at IS NULL
	`
	var args []interface{}
	where := ""

	if filter.CategoryID > 0 {
		where += " AND p.category_id = ?"
		args = append(args, filter.CategoryID)
	}
	if filter.MinPrice > 0 {
		where += " AND p.price >= ?"
		args = append(args, filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		where += " AND p.price <= ?"
		args = append(args, filter.MaxPrice)
	}
	if filter.Search != "" {
		where += " AND (p.name LIKE ? OR p.description LIKE ? OR c.name LIKE ?)"
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// 分页参数
	args = append(args, pagination.PageSize, (pagination.Page-1)*pagination.PageSize)

	// 执行查询
	rows, err := database.DB.Query(query+where+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var products []models.ProductDetail
	for rows.Next() {
		var p models.ProductDetail
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID,
			&p.SKU, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt, &p.CategoryName,
		); err != nil {
			log.Printf("Error scanning product: %v", err)
			continue
		}
		products = append(products, p)
	}

	// 获取总数
	var total int
	countQuery := "SELECT COUNT(*) FROM products p WHERE p.deleted_at IS NULL" + where
	err = database.DB.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total count"})
		return
	}

	totalPages := utils.CalculateTotalPages(total, pagination.PageSize)

	response := models.ProductResponse{
		Products:  products,
		Total:     total,
		Page:      pagination.Page,
		PageSize:  pagination.PageSize,
		TotalPage: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

func UpdateProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 开始事务
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start transaction"})
		return
	}

	// 更新产品
	_, err = tx.Exec(`
		UPDATE products 
		SET name = ?, description = ?, price = ?, stock = ?, 
		    category_id = ?, sku = ?, image_url = ?, updated_at = NOW()
		WHERE id = ?
	`,
		product.Name, product.Description, product.Price, product.Stock,
		product.CategoryID, product.SKU, product.ImageURL, productID)

	if err != nil {
		err := tx.Rollback()
		if err != nil {
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated"})
}

func DeleteProduct(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// 软删除
	_, err = database.DB.Exec(`
		UPDATE products 
		SET deleted_at = NOW() 
		WHERE id = ?
	`, productID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}

func AddProductImage(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var image models.ProductImage
	if err := c.ShouldBindJSON(&image); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证产品是否存在
	var exists bool
	err = database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM products WHERE id = ? AND deleted_at IS NULL)",
		productID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// 插入图片
	result, err := database.DB.Exec(`
		INSERT INTO product_images (product_id, image_url, is_primary)
		VALUES (?, ?, ?)
	`, productID, image.ImageURL, image.IsPrimary)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add image"})
		return
	}

	imageID, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": imageID})
}

func AddProductAttribute(c *gin.Context) {
	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var attribute models.ProductAttribute
	if err := c.ShouldBindJSON(&attribute); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证产品是否存在
	var exists bool
	err = database.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM products WHERE id = ? AND deleted_at IS NULL)",
		productID,
	).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// 插入属性
	result, err := database.DB.Exec(`
		INSERT INTO product_attributes (product_id, name, value)
		VALUES (?, ?, ?)
	`, productID, attribute.Name, attribute.Value)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add attribute"})
		return
	}

	attrID, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": attrID})
}
