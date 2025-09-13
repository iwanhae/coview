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

	if err := r.ParseMultipartForm(0); err != nil { // 0 for no limit
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	var errors []string
	for _, fheader := range files {
		file, err := fheader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to open %s: %v", fheader.Filename, err))
			continue
		}

		filename := filepath.Base(fheader.Filename)
		err = archive.UploadZip(filename, file, 0) // No limit
		file.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Upload failed for %s: %v", filename, err))
		}
	}

	if len(errors) > 0 {
		http.Error(w, strings.Join(errors, "; "), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
