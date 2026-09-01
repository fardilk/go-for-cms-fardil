package routes

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fardilk/cms-porto-fardil/config"
	"github.com/fardilk/cms-porto-fardil/models"
	"github.com/gin-gonic/gin"
)

const maxUploadBytes = 10 << 20 // 10 MB

// allowedImageTypes maps the sniffed content type to the extension we store.
// The extension comes from the sniffed bytes, never from the client's filename,
// so a file called evil.svg cannot be written as evil.svg.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadMedia handles POST /api/media. The CMS had an upload component but the
// API had no endpoint, so adding an image was impossible from the panel.
func UploadMedia(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file provided"})
		return
	}
	if fileHeader.Size > maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds 10 MB"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read upload"})
		return
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := src.Read(head)
	contentType := http.DetectContentType(head[:n])
	ext, ok := allowedImageTypes[contentType]
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{
			"error": fmt.Sprintf("Unsupported type %s; allowed: JPEG, PNG, GIF, WebP", contentType),
		})
		return
	}

	if _, err := src.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not rewind upload"})
		return
	}

	width, height := 0, 0
	if cfg, _, err := image.DecodeConfig(src); err == nil {
		width, height = cfg.Width, cfg.Height
	}
	if _, err := src.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not rewind upload"})
		return
	}

	dir := config.UploadDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload directory is not writable"})
		return
	}

	filename := randomName() + ext
	dst, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store file"})
		return
	}
	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		os.Remove(filepath.Join(dir, filename))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not store file"})
		return
	}

	asset := models.MediaAsset{
		Filename:     filename,
		OriginalName: filepath.Base(fileHeader.Filename),
		MimeType:     contentType,
		SizeBytes:    written,
		Width:        width,
		Height:       height,
		AltText:      strings.TrimSpace(c.PostForm("alt_text")),
	}
	if err := config.DB.Create(&asset).Error; err != nil {
		os.Remove(filepath.Join(dir, filename))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not record upload"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"asset": asset, "url": asset.URL()})
}

// ListMedia handles GET /api/media, newest first, for the CMS picker.
func ListMedia(c *gin.Context) {
	var assets []models.MediaAsset
	if err := config.DB.Order("created_at DESC").Limit(200).Find(&assets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
		return
	}
	out := make([]gin.H, 0, len(assets))
	for _, a := range assets {
		out = append(out, gin.H{"asset": a, "url": a.URL()})
	}
	c.JSON(http.StatusOK, out)
}

// DeleteMedia handles DELETE /api/media/:id and removes the file with the row.
func DeleteMedia(c *gin.Context) {
	var asset models.MediaAsset
	if err := config.DB.First(&asset, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}
	if err := config.DB.Delete(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media"})
		return
	}
	os.Remove(filepath.Join(config.UploadDir(), asset.Filename))
	c.Status(http.StatusNoContent)
}

func randomName() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "upload"
	}
	return hex.EncodeToString(b)
}
