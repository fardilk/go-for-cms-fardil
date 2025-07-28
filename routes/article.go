package routes

import (
	"net/http"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/gin-gonic/gin"
)

// CreateArticle handles POST /api/articles
func CreateArticle(c *gin.Context) {
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := config.DB.Create(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create article"})
		return
	}
	c.JSON(http.StatusCreated, article)
}

// GetArticles handles GET /api/articles
func GetArticles(c *gin.Context) {
	var articles []models.Article
	if err := config.DB.Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}
	c.JSON(http.StatusOK, articles)
}

// GetArticle handles GET /api/articles/:id
func GetArticle(c *gin.Context) {
	id := c.Param("id")
	var article models.Article
	if err := config.DB.Preload("Authors").Preload("Categories").Preload("Tags").First(&article, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}
	c.JSON(http.StatusOK, article)
}

// UpdateArticle handles PUT /api/articles/:id
func UpdateArticle(c *gin.Context) {
	id := c.Param("id")
	var article models.Article
	if err := config.DB.First(&article, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}
	var input models.Article
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Update fields (customize as needed)
	article.Title = input.Title
	article.Content = input.Content
	article.UpdatedAt = input.UpdatedAt
	if err := config.DB.Save(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}
	c.JSON(http.StatusOK, article)
}

// DeleteArticle handles DELETE /api/articles/:id
func DeleteArticle(c *gin.Context) {
	id := c.Param("id")
	var article models.Article
	if err := config.DB.First(&article, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}
	article.IsDeleted = true
	if err := config.DB.Save(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move article to trash"})
		return
	}
	c.Status(http.StatusNoContent)
}
