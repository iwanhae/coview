package server

import (
	"fmt"
	"io"
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

	// Use MultipartReader for streaming uploads - doesn't load entire form into memory
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create multipart reader: %v", err), http.StatusBadRequest)
		return
	}

	var errors []string
	var fileCount int

	// Stream through each part in the multipart form
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to read multipart: %v", err))
			break
		}

		// Only process file parts with form name "files"
		if part.FormName() != "files" {
			part.Close()
			continue
		}

		// Get filename from Content-Disposition header
		filename := part.FileName()
		if filename == "" {
			part.Close()
			continue
		}

		// Extract just the base filename for security
		filename = filepath.Base(filename)

		// Stream directly to disk - UploadZip handles the temp file creation
		err = archive.UploadZip(filename, part, 0) // No limit
		part.Close()
		if err != nil {
			errors = append(errors, fmt.Sprintf("Upload failed for %s: %v", filename, err))
		} else {
			fileCount++
		}
	}

	if fileCount == 0 && len(errors) == 0 {
		http.Error(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	if len(errors) > 0 {
		http.Error(w, strings.Join(errors, "; "), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
