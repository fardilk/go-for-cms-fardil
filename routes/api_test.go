package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func setupRouter() *gin.Engine {
	r := gin.Default()
	// Register your routes here
	r.POST("/api/articles", CreateArticle)
	r.GET("/api/articles", GetArticles)
	r.GET("/api/articles/:id", GetArticle)
	r.POST("/api/tags", CreateTag)
	r.GET("/api/tags", GetTags)
	r.GET("/api/tags/:id", GetTag)
	return r
}

func TestCreateArticle(t *testing.T) {
	r := setupRouter()

	article := map[string]interface{}{
		"title":   "Test Article",
		"content": map[string]interface{}{"blocks": []interface{}{}},
		"status":  "Draft",
	}
	body, _ := json.Marshal(article)
	req, _ := http.NewRequest("POST", "/api/articles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 201, w.Code)
}

func TestGetArticles(t *testing.T) {
	r := setupRouter()
	req, _ := http.NewRequest("GET", "/api/articles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestCreateTag(t *testing.T) {
	r := setupRouter()
	tag := map[string]interface{}{
		"name": "TestTag",
		"slug": "testtag",
	}
	body, _ := json.Marshal(tag)
	req, _ := http.NewRequest("POST", "/api/tags", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)
}

func TestGetTags(t *testing.T) {
	r := setupRouter()
	req, _ := http.NewRequest("GET", "/api/tags", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func TestMain(m *testing.M) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	config.DB = db
	// Migrate models
	db.AutoMigrate(&models.Article{}, &models.Tag{}, &models.Category{}, &models.User{})
	// Delete Articles
	// db.Exec("DELETE FROM articles")
    m.Run()
}
