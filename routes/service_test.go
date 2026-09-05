package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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
		&models.Service{}, &models.ServiceHighlight{}, &models.ServiceReason{},
		&models.ServiceStep{}, &models.ServiceOutcome{}, &models.ServiceMetric{},
		&models.ServiceFaq{}, &models.ServicePlan{}, &models.ServiceProof{},
		&models.ServiceSchedule{},
	))
	assert.NoError(t, db.AutoMigrate(
		&models.Service{}, &models.ServiceHighlight{}, &models.ServiceReason{},
		&models.ServiceStep{}, &models.ServiceOutcome{}, &models.ServiceMetric{},
		&models.ServiceFaq{}, &models.ServicePlan{}, &models.ServiceProof{},
		&models.ServiceSchedule{},
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

// The reason PATCH exists: flipping a page live from a list must not take its
// content with it.
func TestPatchKeepsChildRows(t *testing.T) {
	r := setupServiceRouter(t)
	r.PATCH("/api/services/:id", PatchService)

	create := do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "leadership", "category": "training", "title": "Leadership",
		"template": "program", "published": false,
		"steps": []map[string]interface{}{
			{"title": "Modul 1", "meta": "4 jam", "position": 0},
			{"title": "Modul 2", "meta": "4 jam", "position": 1},
		},
		"faqs": []map[string]interface{}{{"question": "Berapa lama?", "answer": "2 hari"}},
	})
	assert.Equal(t, http.StatusCreated, create.Code)

	patch := do(t, r, "PATCH", "/api/services/1", map[string]interface{}{"published": true})
	assert.Equal(t, http.StatusOK, patch.Code)

	var steps, faqs int64
	config.DB.Model(&models.ServiceStep{}).Count(&steps)
	config.DB.Model(&models.ServiceFaq{}).Count(&faqs)
	assert.Equal(t, int64(2), steps, "a status flip must not delete the curriculum")
	assert.Equal(t, int64(1), faqs)

	w := do(t, r, "GET", "/api/services?published=true", nil)
	var services []models.Service
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &services))
	assert.Len(t, services, 1)
	assert.Len(t, services[0].Steps, 2)
}

func TestPatchRejectsEmptyBody(t *testing.T) {
	r := setupServiceRouter(t)
	r.PATCH("/api/services/:id", PatchService)

	do(t, r, "POST", "/api/services", map[string]interface{}{
		"slug": "sales", "category": "training", "title": "Sales", "template": "program",
	})
	assert.Equal(t, http.StatusBadRequest, do(t, r, "PATCH", "/api/services/1", map[string]interface{}{}).Code)
}

// A layout the site cannot render must be refused at the door. Accepted, the
// band would silently not appear and the editor would be left looking for a
// section they had just configured.
func TestSectionLayoutIsValidated(t *testing.T) {
	r := setupServiceRouter(t)

	base := func() map[string]interface{} {
		return map[string]interface{}{
			"slug": "leadership", "category": "training",
			"title": "Leadership", "template": "program",
		}
	}

	bad := []struct {
		name     string
		sections []map[string]interface{}
	}{
		{"unknown key", []map[string]interface{}{{"key": "gallery", "enabled": true}}},
		{"unknown tone", []map[string]interface{}{{"key": "faqs", "tone": "neon", "enabled": true}}},
		{"duplicate key", []map[string]interface{}{
			{"key": "faqs", "enabled": true},
			{"key": "faqs", "enabled": false},
		}},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			payload := base()
			payload["sections"] = tc.sections
			assert.Equal(t, http.StatusBadRequest, do(t, r, "POST", "/api/services", payload).Code)
		})
	}

	// The arrangement the panel actually posts, and the empty case every
	// service created before this feature existed still sends.
	ok := base()
	ok["sections"] = []map[string]interface{}{
		{"key": "intro", "tone": "auto", "enabled": true},
		{"key": "proofs", "title": "Kata Alumni", "tone": "muted", "enabled": true},
		{"key": "faqs", "tone": "auto", "enabled": false},
	}
	assert.Equal(t, http.StatusCreated, do(t, r, "POST", "/api/services", ok).Code)
}

func TestRatingIsBounded(t *testing.T) {
	r := setupServiceRouter(t)
	payload := map[string]interface{}{
		"slug": "sales", "category": "training",
		"title": "Sales", "template": "program",
		"rating_score": 7.4,
	}
	assert.Equal(t, http.StatusBadRequest, do(t, r, "POST", "/api/services", payload).Code)

	payload["rating_score"] = 4.8
	payload["rating_count"] = 96
	w := do(t, r, "POST", "/api/services", payload)
	assert.Equal(t, http.StatusCreated, w.Code)

	var saved models.Service
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	assert.Equal(t, 4.8, saved.RatingScore)
	assert.Equal(t, 96, saved.RatingCount)
}

// Reasons are the "why certify at all" blocks. They replace children wholesale
// like every other group, and the section key has to be accepted by the layout
// validator or the band could never be arranged.
func TestReasonsRoundTripAndAreArrangeable(t *testing.T) {
	r := setupServiceRouter(t)

	payload := map[string]interface{}{
		"slug": "sertifikasi-trainer-bnsp", "category": "training",
		"title": "Sertifikasi Trainer BNSP", "template": "program",
		"sections": []map[string]interface{}{
			{"key": "reasons", "title": "Kenapa Harus Tersertifikasi?", "tone": "muted", "enabled": true},
		},
		"reasons": []map[string]interface{}{
			{
				"position": 0, "icon": "fa-chart-line", "stat": "47%",
				"title": "Naik jenjang karier", "source": "Dari 3.340 responden",
				"body":      "Melaporkan kenaikan jenjang dalam 24 bulan setelah tersertifikasi.",
				"link_href": "/blog/sertifikasi-dan-karier", "link_text": "Baca selengkapnya",
			},
		},
	}

	w := do(t, r, "POST", "/api/services", payload)
	assert.Equal(t, http.StatusCreated, w.Code)

	var created models.Service
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Len(t, created.Reasons, 1)
	assert.Equal(t, "47%", created.Reasons[0].Stat)
	assert.Equal(t, "/blog/sertifikasi-dan-karier", created.Reasons[0].LinkHref)

	// A removed reason must actually disappear, not linger from the old row.
	payload["reasons"] = []map[string]interface{}{}
	assert.Equal(t, http.StatusOK,
		do(t, r, "PUT", "/api/services/"+strconv.Itoa(int(created.ID)), payload).Code)

	var reasons []models.ServiceReason
	assert.NoError(t, config.DB.Where("service_id = ?", created.ID).Find(&reasons).Error)
	assert.Empty(t, reasons)
}
