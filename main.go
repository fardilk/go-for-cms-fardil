package main

import (
	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/fardilk/cms-porto-fardil/routes"
	"github.com/fardilk/cms-porto-fardil/seed"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Init DB
	config.InitDB()

	// Auto migrate user table
	config.DB.AutoMigrate(
		&models.User{},
		&models.Author{},
		&models.Category{},
		&models.Tag{},
		&models.Status{},
		&models.Article{},
	)

	// Migrate all models
	models.Migrate()

	// Seeding data
	seed.SeedCategories()
	seed.SeedTags()
	seed.SeedSampleData()

	// Init Gin router
	r := gin.Default()

	// CORS middleware for frontend
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"POST", "GET", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	}))

	// Test route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Auth route
	r.POST("/login", routes.Login)

	// Protected dashboard route
	r.GET("/dashboard", routes.Dashboard)

	// Protected me route
	r.GET("/me", routes.Me)

	// Categories CRUD routes
	r.POST("/api/categories", routes.CreateCategory)
	r.GET("/api/categories", routes.GetCategories)
	r.GET("/api/categories/:id", routes.GetCategory)
	r.PUT("/api/categories/:id", routes.UpdateCategory)
	r.DELETE("/api/categories/:id", routes.DeleteCategory)

	// Articles CRUD routes
	r.POST("/api/articles", routes.CreateArticle)
	r.GET("/api/articles", routes.GetArticles)
	r.GET("/api/articles/:id", routes.GetArticle)
	r.PUT("/api/articles/:id", routes.UpdateArticle)
	r.DELETE("/api/articles/:id", routes.DeleteArticle)

	// Tags CRUD routes
	r.POST("/api/tags", routes.CreateTag)
	r.GET("/api/tags", routes.GetTags)
	r.GET("/api/tags/:id", routes.GetTag)

	r.OPTIONS("/*path", func(c *gin.Context) {
		c.Status(204)
	})

	// Serve static files
	r.Static("/images", "./static/images")

	// Run server on PORT from .env
	r.Run(":8000") // default port fallback
}
