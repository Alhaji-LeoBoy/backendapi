package handlers

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/middleware"
	"femProjectSqlc/internal/queue"
	"femProjectSqlc/internal/utils"
)

const (
	maxUploadSize = 10 << 20 // 10 MB
	UploadDir     = "./uploads/images"
)

// UploadHandler handles file upload requests
type UploadHandler struct {
	imageQueue *queue.ImageQueue
	logger     *logger.Logger
}

// NewUploadHandler creates a new UploadHandler and ensures the upload directory exists
func NewUploadHandler(imageQueue *queue.ImageQueue, log *logger.Logger) *UploadHandler {
	if err := os.MkdirAll(UploadDir, 0755); err != nil {
		log.Error("Failed to create upload directory", err)
	}
	return &UploadHandler{
		imageQueue: imageQueue,
		logger:     log,
	}
}

// UploadImage handles multipart image uploads.
// POST /uploads/image
// Form field: "image" (file)
// Returns: { "url": "http://host/uploads/images/<filename>", "filename": "<filename>" }
func (h *UploadHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	authUser, err := middleware.RequireUser(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Limit total body size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "file too large — max 10 MB")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, `form field "image" is required`)
		return
	}
	defer file.Close()

	// Sniff MIME type from first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to read file")
		return
	}
	mimeType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(mimeType, "image/") {
		utils.RespondWithError(w, http.StatusBadRequest, "only image files are accepted")
		return
	}

	// Seek back to start before saving
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to process file")
		return
	}

	// Build a unique filename: <timestamp>-<random-hex>.<ext>
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" || ext == "." {
		ext = ".jpg"
	}
	id := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), randomHex(8))
	filename := id + ext
	filePath := filepath.Join(UploadDir, filename)

	// Write file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		h.logger.Error("Failed to create destination file", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to save image")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.logger.Error("Failed to write image data", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to save image")
		return
	}

	// Push a processing job onto the queue
	job := queue.ImageJob{
		ID:        id,
		FilePath:  filePath,
		FileName:  filename,
		UserID:    authUser.ID,
		MimeType:  mimeType,
		SizeBytes: header.Size,
		QueuedAt:  time.Now().UTC(),
	}
	if err := h.imageQueue.Push(r.Context(), job); err != nil {
		// Non-fatal: image is saved; worker just won't pick it up
		h.logger.Error("Failed to enqueue image job", err)
	}

	// Return a relative path so the Flutter client constructs the full URL
	// using its own base URL — avoids http/https scheme mismatches with tunnels.
	imagePath := fmt.Sprintf("/uploads/images/%s", filename)

	h.logger.Info(fmt.Sprintf("Image uploaded: %s by user %d", filename, authUser.ID))

	utils.RespondWithJSON(w, http.StatusCreated, map[string]string{
		"url":      imagePath,
		"filename": filename,
	})
}

// serverBaseURL derives scheme+host from the request so the returned URL
// works on both emulators and physical devices without hard-coding an IP.
// It respects X-Forwarded-Proto so dev tunnels and reverse proxies (which
// terminate TLS before the Go server) return https:// correctly.
func serverBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:8080"
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
