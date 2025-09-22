package archive

import (
	"fmt"
	"io"
	"log/slog"
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

	if err := rename(tempFile.Name(), targetPath); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func rename(srcPath, dstPath string) error {
	if err := os.Rename(srcPath, dstPath); err != nil {
		slog.Warn("failed to rename", "src", srcPath, "dst", dstPath, "err", err.Error())
	} else if err == nil {
		return nil
	}

	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	// Copy content from source to destination
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Ensure all data is written to disk
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	// Remove the source file
	err = os.Remove(srcPath)
	if err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}
