package routes

import (
	"net/http"
	"time"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func withArticleRelations(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Categories").Preload("Tags").Preload("Authors").Preload("Status")
}

// statusID resolves a status name to its row, so callers can send "PUBLISHED"
// instead of having to know the numeric id.
func statusID(name string) uint {
	var status models.Status
	if err := config.DB.Where("name = ?", name).First(&status).Error; err != nil {
		return 0
	}
	return status.ID
}

// CreateArticle handles POST /api/articles
func CreateArticle(c *gin.Context) {
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if article.StatusID == 0 {
		article.StatusID = statusID("DRAFT")
	}
	now := time.Now()
	article.UpdatedAt = &now

	if err := config.DB.Create(&article).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create article"})
		return
	}

	var saved models.Article
	withArticleRelations(config.DB).First(&saved, article.ID)
	c.JSON(http.StatusCreated, saved)
}

// GetArticles handles GET /api/articles.
//
// Trashed articles are excluded. DeleteArticle sets is_deleted, but nothing
// filtered on it, so a deleted article kept appearing in the panel.
func GetArticles(c *gin.Context) {
	tx := withArticleRelations(config.DB).Where("is_deleted = ?", false)

	if c.Query("published") == "true" {
		tx = tx.Where("status_id = ?", statusID("PUBLISHED")).Where("published_at IS NOT NULL")
	}

	var articles []models.Article
	if err := tx.Order("created_at DESC").Find(&articles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch articles"})
		return
	}
	c.JSON(http.StatusOK, articles)
}

// GetArticle handles GET /api/articles/:id, or ?slug= for the public site.
func GetArticle(c *gin.Context) {
	tx := withArticleRelations(config.DB).Where("is_deleted = ?", false)

	if slug := c.Query("slug"); slug != "" {
		tx = tx.Where("slug = ?", slug)
	} else {
		tx = tx.Where("id = ?", c.Param("id"))
	}

	var article models.Article
	if err := tx.First(&article).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}
	c.JSON(http.StatusOK, article)
}

// UpdateArticle handles PUT /api/articles/:id.
//
// Every editable field is written. The previous version copied only the title,
// the content and the timestamp, so a slug, excerpt, meta description, featured
// image, category or tag entered in the editor was discarded on save without
// anything reporting it.
func UpdateArticle(c *gin.Context) {
	var article models.Article
	if err := config.DB.First(&article, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	var input models.Article
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	article.Title = input.Title
	article.Slug = input.Slug
	article.MetaTitle = input.MetaTitle
	article.MetaDescription = input.MetaDescription
	article.FeaturedImage = input.FeaturedImage
	article.AltText = input.AltText
	article.Excerpt = input.Excerpt
	article.CanonicalURL = input.CanonicalURL
	article.ReadingTime = input.ReadingTime
	article.Content = input.Content
	article.IsFeatured = input.IsFeatured
	article.PublishedAt = input.PublishedAt

	if input.StatusID != 0 {
		article.StatusID = input.StatusID
	}

	now := time.Now()
	article.UpdatedAt = &now

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		// Associations are replaced, not merged: the editor sends the complete
		// set, so a removed category has to actually disappear.
		if err := tx.Model(&article).Association("Categories").Replace(input.Categories); err != nil {
			return err
		}
		if err := tx.Model(&article).Association("Tags").Replace(input.Tags); err != nil {
			return err
		}
		return tx.Save(&article).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}

	var saved models.Article
	withArticleRelations(config.DB).First(&saved, article.ID)
	c.JSON(http.StatusOK, saved)
}

// DeleteArticle handles DELETE /api/articles/:id, moving it to the trash.
func DeleteArticle(c *gin.Context) {
	var article models.Article
	if err := config.DB.First(&article, c.Param("id")).Error; err != nil {
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

// PatchArticle handles PATCH /api/articles/:id for status changes made from a
// list, where the caller does not hold the full article and a PUT would blank
// everything it does not know about.
func PatchArticle(c *gin.Context) {
	var article models.Article
	if err := config.DB.First(&article, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
		return
	}

	var body struct {
		Status     *string `json:"status"`
		IsFeatured *bool   `json:"is_featured"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if body.Status != nil {
		id := statusID(*body.Status)
		if id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown status"})
			return
		}
		updates["status_id"] = id
		// Publishing stamps the date the site sorts and displays by.
		if *body.Status == "PUBLISHED" && article.PublishedAt == nil {
			now := time.Now()
			updates["published_at"] = &now
		}
	}
	if body.IsFeatured != nil {
		updates["is_featured"] = *body.IsFeatured
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nothing to update"})
		return
	}

	if err := config.DB.Model(&article).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update article"})
		return
	}

	var saved models.Article
	withArticleRelations(config.DB).First(&saved, article.ID)
	c.JSON(http.StatusOK, saved)
}
