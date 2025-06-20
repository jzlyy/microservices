package main

import (
	"log"
	"net/http"
	"product-service/controllers"
	"product-service/database"
	"product-service/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer database.CloseDB()

	// 创建Gin路由
	r := gin.Default()

	// 应用中间件
	r.Use(middlewares.CORSMiddleware())
	r.Use(middlewares.LoggingMiddleware())
	r.Use(middlewares.PrometheusMiddleware()) // 添加Prometheus中间件

	// 暴露Prometheus指标端点
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 公共路由
	public := r.Group("/api")
	{
		public.GET("/products", controllers.ListProducts)
		public.GET("/products/:id", controllers.GetProduct)
	}

	// 需要认证的路由组
	authGroup := r.Group("/api")
	authGroup.Use(middlewares.AuthMiddleware())
	{
		// 分类管理
		authGroup.POST("/categories", controllers.CreateCategory)

		// 商品管理
		authGroup.POST("/products", controllers.CreateProduct)
		authGroup.PUT("/products/:id", controllers.UpdateProduct)
		authGroup.DELETE("/products/:id", controllers.DeleteProduct)

		// 商品属性管理
		authGroup.POST("/products/:id/images", controllers.AddProductImage)
		authGroup.POST("/products/:id/attributes", controllers.AddProductAttribute)
	}

	// 启动服务器
	port := ":8080"
	log.Printf("Product service starting on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
