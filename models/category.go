package models

import (
	"net/http"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/gin-gonic/gin"
)

// Category represents the category model.
type Category struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"not null;unique"`
	Slug string `gorm:"not null;unique"`
}

func GetCategories(c *gin.Context) {
	var categories []Category // Use Category struct defined in this package
	if err := config.DB.Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func Migrate() {
	config.DB.AutoMigrate(&Article{}, &Author{}, &Category{}, &Tag{}, &Status{})
}
