package server

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/iwanhae/coview/internal/archive"
	"github.com/iwanhae/coview/internal/config"
)

// Handler is the main HTTP handler that routes requests.
func Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/upload" {
		UploadHandler(w, r)
		return
	}

	if r.URL.Path == "/" {
		archive.ListZips(w, r)
		return
	}

	trimmed := strings.TrimLeft(r.URL.Path, "/")
	if trimmed == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 1 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	zipName := parts[0]
	if !strings.HasSuffix(zipName, ".zip") {
		http.NotFound(w, r)
		return
	}

	cfg := config.Get()
	if cfg == nil {
		http.Error(w, "Configuration not loaded", http.StatusInternalServerError)
		return
	}

	zipPath := filepath.Join(cfg.Data.Dir, zipName)

	if len(parts) == 1 {
		archive.ListFilesInZip(w, r, zipName, zipPath)
		return
	}

	if len(parts) >= 2 {
		filePath := strings.Join(parts[1:], "/")
		archive.ServeFileFromZip(w, r, zipName, zipPath, filePath)
		return
	}

	http.NotFound(w, r)
}

// UploadHandler handles ZIP file uploads via POST /upload.
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := config.Get()
	if cfg == nil {
		http.Error(w, "Configuration not loaded", http.StatusInternalServerError)
		return
	}

	r.ParseMultipartForm(cfg.Upload.MaxSize)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := filepath.Base(header.Filename)
	err = archive.UploadZip(filename, file, cfg.Upload.MaxSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("Upload failed: %v", err), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
