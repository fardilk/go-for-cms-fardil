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

func setupArticleRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	for _, m := range []interface{}{&models.Article{}, &models.Status{}, &models.Category{}, &models.Tag{}} {
		assert.NoError(t, config.DB.Migrator().DropTable(m))
	}
	assert.NoError(t, config.DB.AutoMigrate(&models.Article{}, &models.Status{}, &models.Category{}, &models.Tag{}))

	for _, name := range []string{"DRAFT", "PUBLISHED", "ARCHIVED", "DELETED"} {
		assert.NoError(t, config.DB.Create(&models.Status{Name: name}).Error)
	}

	r := gin.New()
	r.POST("/api/articles", CreateArticle)
	r.GET("/api/articles", GetArticles)
	r.GET("/api/articles/:id", GetArticle)
	r.PUT("/api/articles/:id", UpdateArticle)
	r.PATCH("/api/articles/:id", PatchArticle)
	r.DELETE("/api/articles/:id", DeleteArticle)
	return r
}

func call(t *testing.T, r *gin.Engine, method, path string, payload interface{}) *httptest.ResponseRecorder {
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

// The API used to answer with Go field names, so every consumer looking for
// `title` found nothing and the panel's list filtered the whole payload away.
func TestArticleJSONUsesSnakeCaseKeys(t *testing.T) {
	r := setupArticleRouter(t)

	w := call(t, r, "POST", "/api/articles", map[string]any{
		"title": "Kepemimpinan", "slug": "kepemimpinan",
		"featured_image": "/images/a.png", "meta_description": "ringkas",
	})
	assert.Equal(t, http.StatusCreated, w.Code)

	var raw map[string]any
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &raw))
	for _, key := range []string{"title", "slug", "featured_image", "meta_description", "content"} {
		assert.Contains(t, raw, key)
	}
	assert.NotContains(t, raw, "Title")
	assert.NotContains(t, raw, "FeaturedImage")
}

// Saving used to copy only the title, the content and the timestamp, silently
// discarding everything else the editor had collected.
func TestUpdateKeepsEveryEditableField(t *testing.T) {
	r := setupArticleRouter(t)

	create := call(t, r, "POST", "/api/articles", map[string]any{"title": "Draf", "slug": "draf"})
	assert.Equal(t, http.StatusCreated, create.Code)

	update := call(t, r, "PUT", "/api/articles/1", map[string]any{
		"title":            "Judul Final",
		"slug":             "judul-final",
		"excerpt":          "Ringkasan singkat",
		"meta_description": "Deskripsi untuk mesin pencari",
		"featured_image":   "/images/hero.png",
		"alt_text":         "Foto pelatihan",
		"reading_time":     6,
		"is_featured":      true,
		"content":          map[string]any{"blocks": []any{map[string]any{"type": "paragraph", "data": map[string]any{"text": "Halo"}}}},
	})
	assert.Equal(t, http.StatusOK, update.Code)

	var saved models.Article
	assert.NoError(t, json.Unmarshal(update.Body.Bytes(), &saved))
	assert.Equal(t, "judul-final", saved.Slug)
	assert.Equal(t, "Ringkasan singkat", saved.Excerpt)
	assert.Equal(t, "Deskripsi untuk mesin pencari", saved.MetaDescription)
	assert.Equal(t, "/images/hero.png", saved.FeaturedImage)
	assert.Equal(t, "Foto pelatihan", saved.AltText)
	assert.Equal(t, 6, saved.ReadingTime)
	assert.True(t, saved.IsFeatured)
	assert.Contains(t, string(saved.Content), "paragraph")
}

// Delete is a soft delete, but nothing filtered on the flag, so a trashed
// article stayed visible in the panel.
func TestDeletedArticleLeavesTheList(t *testing.T) {
	r := setupArticleRouter(t)

	call(t, r, "POST", "/api/articles", map[string]any{"title": "Dibuang", "slug": "dibuang"})
	call(t, r, "POST", "/api/articles", map[string]any{"title": "Disimpan", "slug": "disimpan"})

	assert.Equal(t, http.StatusNoContent, call(t, r, "DELETE", "/api/articles/1", nil).Code)

	w := call(t, r, "GET", "/api/articles", nil)
	var list []models.Article
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Len(t, list, 1)
	assert.Equal(t, "Disimpan", list[0].Title)

	assert.Equal(t, http.StatusNotFound, call(t, r, "GET", "/api/articles/1", nil).Code)
}

func TestPublishStampsTheDate(t *testing.T) {
	r := setupArticleRouter(t)
	call(t, r, "POST", "/api/articles", map[string]any{"title": "Terbit", "slug": "terbit"})

	w := call(t, r, "PATCH", "/api/articles/1", map[string]any{"status": "PUBLISHED"})
	assert.Equal(t, http.StatusOK, w.Code)

	var saved models.Article
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	assert.NotNil(t, saved.PublishedAt, "publishing must record when it happened")

	published := call(t, r, "GET", "/api/articles?published=true", nil)
	var list []models.Article
	assert.NoError(t, json.Unmarshal(published.Body.Bytes(), &list))
	assert.Len(t, list, 1)
}

func TestDraftIsExcludedFromThePublishedFeed(t *testing.T) {
	r := setupArticleRouter(t)
	call(t, r, "POST", "/api/articles", map[string]any{"title": "Masih draf", "slug": "masih-draf"})

	w := call(t, r, "GET", "/api/articles?published=true", nil)
	var list []models.Article
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Empty(t, list, "the site must never build a draft into the blog")
}

func TestArticleLookupBySlug(t *testing.T) {
	r := setupArticleRouter(t)
	call(t, r, "POST", "/api/articles", map[string]any{"title": "Cari Saya", "slug": "cari-saya"})

	w := call(t, r, "GET", "/api/articles/0?slug=cari-saya", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var saved models.Article
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &saved))
	assert.Equal(t, "Cari Saya", saved.Title)
}
