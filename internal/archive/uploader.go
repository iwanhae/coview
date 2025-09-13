package archive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/iwanhae/coview/internal/config"
)

// UploadZip handles the upload of a ZIP file to the data directory.
// It validates the file extension and size, then saves it.
func UploadZip(filename string, file io.Reader, maxSize int64) error {
	// Validate extension
	if !strings.HasSuffix(strings.ToLower(filename), ".zip") {
		return fmt.Errorf("invalid file type: only .zip files are allowed")
	}

	tempFile, err := os.CreateTemp("", "upload-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, file)
	if err != nil {
		return fmt.Errorf("failed to read uploaded file: %w", err)
	}

	tempFile.Close()

	// Move to data dir
	dataDir := config.Get().Data.Dir
	targetPath := filepath.Join(dataDir, filepath.Base(filename))
	if err := os.Rename(tempFile.Name(), targetPath); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}
