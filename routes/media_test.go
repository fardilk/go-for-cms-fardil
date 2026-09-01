package routes

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
)

func setupMediaRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	assert.NoError(t, config.DB.Migrator().DropTable(&models.MediaAsset{}))
	assert.NoError(t, config.DB.AutoMigrate(&models.MediaAsset{}))

	dir := t.TempDir()
	t.Setenv("UPLOAD_DIR", dir)

	r := gin.New()
	r.POST("/api/media", UploadMedia)
	r.GET("/api/media/usage", MediaUsage)
	return r, dir
}

// pngBytes builds a real PNG so the handler's content sniffing sees an image.
func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	assert.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func uploadRequest(t *testing.T, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	assert.NoError(t, err)
	_, err = part.Write(content)
	assert.NoError(t, err)
	assert.NoError(t, mw.Close())

	req, err := http.NewRequest("POST", "/api/media", &body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadStoresImageAndRecordsDimensions(t *testing.T) {
	r, dir := setupMediaRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, uploadRequest(t, "photo.png", pngBytes(t, 24, 12)))
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Asset models.MediaAsset `json:"asset"`
		URL   string            `json:"url"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 24, resp.Asset.Width)
	assert.Equal(t, 12, resp.Asset.Height)
	assert.Equal(t, "image/png", resp.Asset.MimeType)

	entries, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1, "the file must actually land on disk")
}

// The stored extension comes from the sniffed bytes, never the client filename,
// so a mislabelled upload cannot choose what it is written as.
func TestUploadIgnoresClientSuppliedExtension(t *testing.T) {
	r, _ := setupMediaRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, uploadRequest(t, "payload.svg", pngBytes(t, 4, 4)))
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		URL string `json:"url"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.URL, ".png")
	assert.NotContains(t, resp.URL, ".svg")
}

func TestUploadRejectsNonImage(t *testing.T) {
	r, _ := setupMediaRouter(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, uploadRequest(t, "notes.txt", []byte("this is definitely not an image at all")))
	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

// The upload directory is a fixed-size volume, so the budget must refuse the
// write before the filesystem does.
func TestUploadRefusedWhenQuotaExceeded(t *testing.T) {
	r, _ := setupMediaRouter(t)
	t.Setenv("MEDIA_QUOTA_BYTES", "1")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, uploadRequest(t, "photo.png", pngBytes(t, 32, 32)))
	assert.Equal(t, http.StatusInsufficientStorage, w.Code)
	assert.Contains(t, w.Body.String(), "Media storage is full")
}

func TestUsageReportsQuota(t *testing.T) {
	r, _ := setupMediaRouter(t)
	t.Setenv("MEDIA_QUOTA_BYTES", "1048576")

	r.ServeHTTP(httptest.NewRecorder(), uploadRequest(t, "photo.png", pngBytes(t, 8, 8)))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/media/usage", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	var usage struct {
		UsedBytes  int64 `json:"used_bytes"`
		QuotaBytes int64 `json:"quota_bytes"`
		FileCount  int64 `json:"file_count"`
	}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &usage))
	assert.Equal(t, int64(1048576), usage.QuotaBytes)
	assert.Equal(t, int64(1), usage.FileCount)
	assert.Greater(t, usage.UsedBytes, int64(0))
}
