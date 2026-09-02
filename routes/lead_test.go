package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func setupLeadRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	assert.NoError(t, config.DB.Migrator().DropTable(&models.Lead{}))
	assert.NoError(t, config.DB.AutoMigrate(&models.Lead{}))

	r := gin.New()
	r.POST("/api/leads", CreateLead)
	r.GET("/api/leads", GetLeads)
	r.PATCH("/api/leads/:id", UpdateLead)
	r.DELETE("/api/leads/:id", DeleteLead)
	return r
}

func postLead(t *testing.T, r *gin.Engine, payload map[string]any, ip string) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(payload)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "/api/leads", bytes.NewBuffer(b))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	// Distinct client per test, so the shared limiter does not leak between them.
	req.Header.Set("X-Forwarded-For", ip)
	req.RemoteAddr = ip + ":1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func validLead() map[string]any {
	return map[string]any{
		"name":        "Budi Santoso",
		"email":       "budi@perusahaan.co.id",
		"phone":       "081234567890",
		"company":     "PT Contoh",
		"message":     "Kami tertarik program leadership untuk 20 supervisor.",
		"source_path": "/services/training/leadership",
	}
}

func countLeads(t *testing.T) int64 {
	t.Helper()
	var n int64
	config.DB.Model(&models.Lead{}).Count(&n)
	return n
}

func TestLeadIsStoredWithSubmittedFields(t *testing.T) {
	r := setupLeadRouter(t)

	w := postLead(t, r, validLead(), "10.0.0.1")
	assert.Equal(t, http.StatusCreated, w.Code)

	var stored models.Lead
	assert.NoError(t, config.DB.First(&stored).Error)
	assert.Equal(t, "Budi Santoso", stored.Name)
	assert.Equal(t, "081234567890", stored.Phone)
	assert.Equal(t, "PT Contoh", stored.Company)
	assert.Equal(t, "/services/training/leadership", stored.SourcePath)
	assert.Equal(t, models.LeadNew, stored.Status)
}

// The honeypot must look like success to whatever filled it, while storing
// nothing at all.
func TestHoneypotIsSilentlyDropped(t *testing.T) {
	r := setupLeadRouter(t)

	payload := validLead()
	payload["website"] = "http://spam.example"

	w := postLead(t, r, payload, "10.0.0.2")
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, int64(0), countLeads(t), "a honeypot hit must not be stored")
}

func TestRequiredFieldsAreReported(t *testing.T) {
	r := setupLeadRouter(t)

	w := postLead(t, r, map[string]any{"company": "PT Contoh"}, "10.0.0.3")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var body struct {
		Fields map[string]string `json:"fields"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, f := range []string{"name", "email", "phone", "message"} {
		assert.Contains(t, body.Fields, f)
	}
	assert.Equal(t, int64(0), countLeads(t))
}

func TestInvalidEmailIsRejected(t *testing.T) {
	r := setupLeadRouter(t)

	payload := validLead()
	payload["email"] = "bukan-email"

	w := postLead(t, r, payload, "10.0.0.4")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "email")
	assert.Equal(t, int64(0), countLeads(t))
}

func TestRateLimitStopsFloodFromOneClient(t *testing.T) {
	r := setupLeadRouter(t)

	accepted := 0
	var lastCode int
	for i := 0; i < 8; i++ {
		w := postLead(t, r, validLead(), "10.0.0.5")
		lastCode = w.Code
		if w.Code == http.StatusCreated {
			accepted++
		}
	}

	assert.Equal(t, 5, accepted, "the limiter allows five per window")
	assert.Equal(t, http.StatusTooManyRequests, lastCode)
	assert.Equal(t, int64(5), countLeads(t))
}

func TestStatusUpdateRejectsUnknownValue(t *testing.T) {
	r := setupLeadRouter(t)
	assert.Equal(t, http.StatusCreated, postLead(t, r, validLead(), "10.0.0.6").Code)

	body, _ := json.Marshal(map[string]any{"status": "maybe"})
	req, _ := http.NewRequest("PATCH", "/api/leads/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestListReportsUnreadCount(t *testing.T) {
	r := setupLeadRouter(t)
	assert.Equal(t, http.StatusCreated, postLead(t, r, validLead(), "10.0.0.7").Code)
	assert.Equal(t, http.StatusCreated, postLead(t, r, validLead(), "10.0.0.8").Code)

	body, _ := json.Marshal(map[string]any{"status": models.LeadContacted})
	req, _ := http.NewRequest("PATCH", "/api/leads/1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/leads", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var list struct {
		Items    []models.Lead `json:"items"`
		Total    int64         `json:"total"`
		NewCount int64         `json:"new_count"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Equal(t, int64(2), list.Total)
	assert.Equal(t, int64(1), list.NewCount, "only the untouched lead counts as unread")
}

// A registration is a promise to print a certificate and post it, so the
// fields that decide what is printed and where it goes are all required —
// while a plain enquiry must stay submittable with only a message.
func TestRegistrationRequiresCertificateFields(t *testing.T) {
	r := setupLeadRouter(t)

	w := postLead(t, r, map[string]any{
		"kind":  models.KindRegistration,
		"name":  "Budi Santoso",
		"email": "budi@perusahaan.co.id",
		"phone": "081200000000",
	}, "203.0.113.90")

	assert.Equal(t, 400, w.Code)

	var body struct {
		Fields map[string]string `json:"fields"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, field := range []string{
		"company", "company_address", "division", "position", "city",
		"certificate_address", "referral_source",
	} {
		assert.Contains(t, body.Fields, field)
	}
	// The registration form does not ask for one, so demanding it would make
	// every submission fail.
	assert.NotContains(t, body.Fields, "message")
}

func TestRegistrationIsStoredWithItsOwnFields(t *testing.T) {
	r := setupLeadRouter(t)

	w := postLead(t, r, map[string]any{
		"kind":                models.KindRegistration,
		"name":                "Budi Santoso",
		"email":               "budi@perusahaan.co.id",
		"phone":               "081200000000",
		"company":             "PT Contoh Nusantara",
		"company_address":     "Jl. Sudirman No. 1, Jakarta",
		"division":            "Human Capital",
		"position":            "Learning Manager",
		"city":                "Bekasi",
		"certificate_address": "Jl. Melati No. 7, Bekasi",
		"referral_source":     "instagram",
		"source_path":         "/registration?program=sertifikasi-trainer-bnsp",
	}, "203.0.113.91")

	assert.Equal(t, 201, w.Code)

	var stored models.Lead
	assert.NoError(t, config.DB.Order("id DESC").First(&stored).Error)
	assert.Equal(t, models.KindRegistration, stored.Kind)
	assert.Equal(t, "Budi Santoso", stored.Name)
	assert.Equal(t, "Jl. Melati No. 7, Bekasi", stored.CertificateAddress)
	assert.Equal(t, "instagram", stored.ReferralSource)
	assert.Equal(t, "Learning Manager", stored.Position)
	assert.Equal(t, models.LeadNew, stored.Status)
}

// The contact form sends no kind at all, and must keep working exactly as it
// did before registrations existed.
func TestEnquiryWithoutKindStillNeedsAMessage(t *testing.T) {
	r := setupLeadRouter(t)

	w := postLead(t, r, map[string]any{
		"name":  "Sari",
		"email": "sari@contoh.id",
		"phone": "081200000001",
	}, "203.0.113.92")

	assert.Equal(t, 400, w.Code)

	var body struct {
		Fields map[string]string `json:"fields"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Contains(t, body.Fields, "message")
	assert.NotContains(t, body.Fields, "certificate_address")
}
