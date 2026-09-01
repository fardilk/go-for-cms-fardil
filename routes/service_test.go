package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func setupServiceRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.Migrator().DropTable(
		&models.Service{}, &models.ServiceHighlight{}, &models.ServiceStep{},
		&models.ServiceOutcome{}, &models.ServiceMetric{}, &models.ServiceFaq{},
		&models.ServicePlan{}, &models.ServiceProof{}, &models.ServiceSchedule{},
	))
	assert.NoError(t, db.AutoMigrate(
		&models.Service{}, &models.ServiceHighlight{}, &models.ServiceStep{},
		&models.ServiceOutcome{}, &models.ServiceMetric{}, &models.ServiceFaq{},
		&models.ServicePlan{}, &models.ServiceProof{}, &models.ServiceSchedule{},
	))
	config.DB = db

	r := gin.New()
	r.GET("/api/services", GetServices)
	r.GET("/api/services/:id", GetService)
	r.POST("/api/services", CreateService)
	r.PUT("/api/services/:id", UpdateService)
	r.DELETE("/api/services/:id", DeleteService)
	return r
}

func do(t *testing.T, r *gin.Engine, method, path string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Buffer
	if payload != nil {
		b, err := json.Marshal(payload)
		assert.NoError(t, err)
		body = bytes.NewBuffer(b)
	} else {
		body = bytes.NewBuffer(nil)
	}
	req, err := http.NewRequest(method, path, body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestServiceRejectsUnknownTemplate(t *testing.T) {
	r := setupServiceRouter(t)
	w := do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "leadership", "category": "training",
		"title": "Leadership", "template": "brochure",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServiceRejectsMissingSlug(t *testing.T) {
	r := setupServiceRouter(t)
	w := do(t, r, "POST", "/api/services", map[string]interface{}{
		"category": "training", "title": "Leadership", "template": "program",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// The update path replaces children wholesale. This is the part worth guarding:
// a removed step must actually disappear rather than linger from the old row.
func TestUpdateReplacesChildrenInsteadOfAppending(t *testing.T) {
	r := setupServiceRouter(t)

	create := do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "leadership", "category": "training", "title": "Leadership",
		"template": "program",
		"steps": []map[string]interface{}{
			{"title": "Module 1", "meta": "4 jam", "position": 0},
			{"title": "Module 2", "meta": "4 jam", "position": 1},
			{"title": "Module 3", "meta": "4 jam", "position": 2},
		},
	})
	assert.Equal(t, http.StatusCreated, create.Code)

	var created models.Service
	assert.NoError(t, json.Unmarshal(create.Body.Bytes(), &created))
	assert.Len(t, created.Steps, 3)

	update := do(t, r, "PUT", "/api/services/1", map[string]interface{}{
		"slug": "leadership", "category": "training", "title": "Leadership",
		"template": "program",
		"steps": []map[string]interface{}{
			{"title": "Module 1 revised", "meta": "6 jam", "position": 0},
		},
	})
	assert.Equal(t, http.StatusOK, update.Code)

	var updated models.Service
	assert.NoError(t, json.Unmarshal(update.Body.Bytes(), &updated))
	assert.Len(t, updated.Steps, 1, "old steps must be deleted, not kept alongside the new one")
	assert.Equal(t, "Module 1 revised", updated.Steps[0].Title)

	var orphans int64
	config.DB.Model(&models.ServiceStep{}).Count(&orphans)
	assert.Equal(t, int64(1), orphans, "no orphaned step rows may survive the update")
}

func TestPublishedFilterHidesDrafts(t *testing.T) {
	r := setupServiceRouter(t)

	do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "draft-one", "category": "training", "title": "Draft",
		"template": "program", "published": false,
	})
	do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "live-one", "category": "training", "title": "Live",
		"template": "program", "published": true,
	})

	w := do(t, r, "GET", "/api/services?published=true", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var services []models.Service
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &services))
	assert.Len(t, services, 1)
	assert.Equal(t, "live-one", services[0].Slug)
}

func TestDeleteRemovesChildren(t *testing.T) {
	r := setupServiceRouter(t)

	do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "sales", "category": "training", "title": "Sales", "template": "program",
		"faqs": []map[string]interface{}{{"question": "Berapa lama?", "answer": "2 hari"}},
	})

	assert.Equal(t, http.StatusNoContent, do(t, r, "DELETE", "/api/services/1", nil).Code)

	var faqs int64
	config.DB.Model(&models.ServiceFaq{}).Count(&faqs)
	assert.Equal(t, int64(0), faqs, "child rows must not outlive the service")
}
