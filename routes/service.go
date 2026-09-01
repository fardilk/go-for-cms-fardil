package routes

import (
	"net/http"
	"slices"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// withServiceChildren preloads every repeating group in display order.
func withServiceChildren(tx *gorm.DB) *gorm.DB {
	order := func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }
	return tx.
		Preload("Highlights", order).
		Preload("Steps", order).
		Preload("Outcomes", order).
		Preload("Metrics", order).
		Preload("Faqs", order).
		Preload("Plans", order).
		Preload("Proofs", order).
		Preload("Schedules", order)
}

// GetServices handles GET /api/services. The public site build passes
// ?published=true so drafts never reach production.
func GetServices(c *gin.Context) {
	tx := withServiceChildren(config.DB)
	if c.Query("published") == "true" {
		tx = tx.Where("published = ?", true)
	}
	if cat := c.Query("category"); cat != "" {
		tx = tx.Where("category = ?", cat)
	}

	var services []models.Service
	if err := tx.Order("category ASC, sort_order ASC, id ASC").Find(&services).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}
	c.JSON(http.StatusOK, services)
}

// GetService handles GET /api/services/:id, accepting either the numeric id or
// a "category/slug" pair via query params.
func GetService(c *gin.Context) {
	var service models.Service
	tx := withServiceChildren(config.DB)

	if cat, slug := c.Query("category"), c.Query("slug"); cat != "" && slug != "" {
		tx = tx.Where("category = ? AND slug = ?", cat, slug)
	} else {
		tx = tx.Where("id = ?", c.Param("id"))
	}

	if err := tx.First(&service).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}
	c.JSON(http.StatusOK, service)
}

// CreateService handles POST /api/services.
func CreateService(c *gin.Context) {
	var service models.Service
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if msg, ok := validateService(&service); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	if err := config.DB.Create(&service).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}
	c.JSON(http.StatusCreated, service)
}

// UpdateService handles PUT /api/services/:id.
//
// Children are replaced wholesale rather than diffed: the editor posts the full
// ordered list every time, so deleting the old rows and inserting the new ones
// is both simpler and the only way a removed row actually disappears.
func UpdateService(c *gin.Context) {
	var existing models.Service
	if err := config.DB.First(&existing, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	var incoming models.Service
	if err := c.ShouldBindJSON(&incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if msg, ok := validateService(&incoming); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	incoming.ID = existing.ID

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := deleteServiceChildren(tx, existing.ID); err != nil {
			return err
		}
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).
			Clauses(clause.OnConflict{UpdateAll: true}).
			Save(&incoming).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	var saved models.Service
	withServiceChildren(config.DB).First(&saved, existing.ID)
	c.JSON(http.StatusOK, saved)
}

// DeleteService handles DELETE /api/services/:id.
func DeleteService(c *gin.Context) {
	id := c.Param("id")
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var service models.Service
		if err := tx.First(&service, id).Error; err != nil {
			return err
		}
		if err := deleteServiceChildren(tx, service.ID); err != nil {
			return err
		}
		return tx.Delete(&service).Error
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func deleteServiceChildren(tx *gorm.DB, serviceID uint) error {
	children := []interface{}{
		&models.ServiceHighlight{}, &models.ServiceStep{}, &models.ServiceOutcome{},
		&models.ServiceMetric{}, &models.ServiceFaq{}, &models.ServicePlan{},
		&models.ServiceProof{}, &models.ServiceSchedule{},
	}
	for _, child := range children {
		if err := tx.Where("service_id = ?", serviceID).Delete(child).Error; err != nil {
			return err
		}
	}
	return nil
}

func validateService(s *models.Service) (string, bool) {
	if s.Slug == "" || s.Category == "" {
		return "slug and category are required", false
	}
	if s.Title == "" {
		return "title is required", false
	}
	if !slices.Contains(models.ValidTemplates, s.Template) {
		return "template must be one of: program, engagement, retainer", false
	}
	return "", true
}
