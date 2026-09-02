package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

// The dashboard reads these numbers, so a lead that never gets counted, or a
// day that quietly disappears from the trend, is a wrong report rather than a
// visible failure. Count them here.
func TestGetStats(t *testing.T) {
	db := config.DB
	db.AutoMigrate(&models.Lead{}, &models.Service{}, &models.ServiceSchedule{}, &models.MediaAsset{})
	db.Exec("DELETE FROM leads")
	db.Exec("DELETE FROM service_schedules")
	db.Exec("DELETE FROM services")

	now := time.Now()
	seed := []models.Lead{
		{Name: "a", Email: "a@x.id", Phone: "1", Message: "m", Status: models.LeadNew,
			SourcePath: "/home/contact?program=sertifikasi-trainer-bnsp", CreatedAt: now},
		{Name: "b", Email: "b@x.id", Phone: "2", Message: "m", Status: models.LeadNew,
			SourcePath: "/home/contact?program=sertifikasi-trainer-bnsp", CreatedAt: now.AddDate(0, 0, -3)},
		{Name: "c", Email: "c@x.id", Phone: "3", Message: "m", Status: models.LeadWon,
			SourcePath: "/home/contact", CreatedAt: now.AddDate(0, 0, -20)},
	}
	for i := range seed {
		assert.NoError(t, db.Create(&seed[i]).Error)
	}

	service := models.Service{Slug: "sertifikasi-trainer-bnsp", Category: "training", Title: "BNSP", Published: true}
	assert.NoError(t, db.Create(&service).Error)
	starts := now.AddDate(0, 0, 13)
	assert.NoError(t, db.Create(&models.ServiceSchedule{
		ServiceID: service.ID, StartsAt: &starts, City: "Zoom", Format: "online",
		SeatsTotal: 20, SeatsLeft: 18,
	}).Error)

	r := gin.New()
	r.GET("/api/stats", GetStats)
	req, _ := http.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var got statsResponse
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))

	assert.EqualValues(t, 3, got.Leads.Total)
	assert.EqualValues(t, 2, got.Leads.New)
	assert.EqualValues(t, 2, got.Leads.Last7)
	assert.EqualValues(t, 3, got.Leads.Last30)

	// The trend always spans the full window, empty days included.
	assert.Len(t, got.Leads.ByDay, trendDays)
	var summed int64
	for _, d := range got.Leads.ByDay {
		summed += d.Count
	}
	assert.EqualValues(t, 3, summed)

	// A programme link is attributable; a plain visit is not.
	pages := map[string]int64{}
	for _, p := range got.Leads.ByPage {
		pages[p.Key] = p.Count
	}
	assert.EqualValues(t, 2, pages["/home/contact?program=sertifikasi-trainer-bnsp"])
	assert.EqualValues(t, 1, pages["/home/contact"])

	assert.Len(t, got.Upcoming, 1)
	assert.Equal(t, "BNSP", got.Upcoming[0].Service)
	assert.EqualValues(t, 18, got.Upcoming[0].SeatsLeft)
	assert.EqualValues(t, 2, got.Upcoming[0].Leads, "batch should count the leads that named it")
}
