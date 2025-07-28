package routes

import (
	"net/http"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/gin-gonic/gin"
)

// CreateTag handles POST /api/tags
func CreateTag(c *gin.Context) {
	var tag models.Tag
	if err := c.ShouldBindJSON(&tag); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := config.DB.Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}
	c.JSON(http.StatusCreated, tag)
}

// GetTags handles GET /api/tags
func GetTags(c *gin.Context) {
	var tags []models.Tag
	if err := config.DB.Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

// GetTag handles GET /api/tags/:id
func GetTag(c *gin.Context) {
	id := c.Param("id")
	var tag models.Tag
	if err := config.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}
	c.JSON(http.StatusOK, tag)
}
