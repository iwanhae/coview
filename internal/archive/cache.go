package archive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a cached ZIP file metadata
type CacheEntry struct {
	Info    ZipInfo   `json:"info"`
	ModTime time.Time `json:"mod_time"`
	Size    int64     `json:"size"`
}

// ZipCacheManager manages file-based caching for ZIP metadata
type ZipCacheManager struct {
	mu       sync.RWMutex
	cacheDir string
}

var (
	cacheManager     *ZipCacheManager
	cacheManagerOnce sync.Once
)

// GetCacheManager returns the singleton cache manager instance
func GetCacheManager() *ZipCacheManager {
	cacheManagerOnce.Do(func() {
		// Use /tmp directory for Kubernetes pod ephemeral storage
		cacheDir := "/tmp/coview-cache"
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			// Fallback to system temp dir if /tmp fails
			cacheDir = filepath.Join(os.TempDir(), "coview-cache")
			os.MkdirAll(cacheDir, 0755)
		}
		cacheManager = &ZipCacheManager{
			cacheDir: cacheDir,
		}
	})
	return cacheManager
}

// getCacheFilePath returns the cache file path for a given ZIP file
func (cm *ZipCacheManager) getCacheFilePath(zipPath string) string {
	// Use base name and hash for cache file name to avoid path issues
	baseName := filepath.Base(zipPath)
	return filepath.Join(cm.cacheDir, baseName+".cache.json")
}

// Get retrieves cached metadata for a ZIP file
// Returns nil if cache is invalid or doesn't exist
func (cm *ZipCacheManager) Get(zipPath string, currentModTime time.Time, currentSize int64) *ZipInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	cacheFile := cm.getCacheFilePath(zipPath)
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		// Cache file doesn't exist or can't be read
		return nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Invalid cache file
		return nil
	}

	// Validate cache based on ModTime and Size
	if !entry.ModTime.Equal(currentModTime) || entry.Size != currentSize {
		// Cache is stale
		return nil
	}

	return &entry.Info
}

// Set stores metadata for a ZIP file in cache
func (cm *ZipCacheManager) Set(zipPath string, info ZipInfo, modTime time.Time, size int64) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entry := CacheEntry{
		Info:    info,
		ModTime: modTime,
		Size:    size,
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cacheFile := cm.getCacheFilePath(zipPath)
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Invalidate removes a cache entry for a specific ZIP file
func (cm *ZipCacheManager) Invalidate(zipPath string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cacheFile := cm.getCacheFilePath(zipPath)
	err := os.Remove(cacheFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	return nil
}

// Clear removes all cache files
func (cm *ZipCacheManager) Clear() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entries, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(cm.cacheDir, entry.Name())
			if err := os.Remove(path); err != nil {
				// Log but continue with other files
				continue
			}
		}
	}

	return nil
}

// GetStats returns cache statistics
func (cm *ZipCacheManager) GetStats() (totalFiles int, totalSize int64, err error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	entries, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				totalFiles++
				totalSize += info.Size()
			}
		}
	}

	return totalFiles, totalSize, nil
}
