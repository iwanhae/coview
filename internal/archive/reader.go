package archive

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iwanhae/coview/internal/config"
	"github.com/iwanhae/coview/pkg/natural"
)

type ZipInfo struct {
	Name       string
	Size       int64
	ImageCount int
	SizeStr    string
	FirstImage string
}

// ListZips lists all ZIP files in the data directory and renders the template.
func ListZips(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	if cfg == nil {
		http.Error(w, "Configuration not loaded", http.StatusInternalServerError)
		return
	}

	files, err := filepath.Glob(filepath.Join(cfg.Data.Dir, "*.zip"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var bases []string
	for _, f := range files {
		bases = append(bases, filepath.Base(f))
	}
	sort.Sort(natural.StringSlice(bases))

	var zipInfos []ZipInfo
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, base := range bases {
		fullpath := filepath.Join(cfg.Data.Dir, base)
		stat, err := os.Stat(fullpath)
		if err != nil {
			continue
		}
		size := stat.Size()

		rdr, err := zip.OpenReader(fullpath)
		if err != nil {
			continue
		}
		var files natural.StringSlice
		for _, f := range rdr.File {
			name := strings.ToLower(f.Name)
			for _, ext := range imageExts {
				if strings.HasSuffix(name, ext) {
					files = append(files, f.Name)
					break
				}
			}
		}
		rdr.Close()

		sort.Sort(files)
		count := len(files)
		firstImage := ""
		if count > 0 {
			firstImage = files[0]
		}

		zipInfos = append(zipInfos, ZipInfo{
			Name:       base,
			Size:       size,
			ImageCount: count,
			SizeStr:    fmt.Sprintf("%.1f MB", float64(size)/(1024*1024)),
			FirstImage: firstImage,
		})
	}

	t, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, zipInfos)
}

// ListFilesInZip lists the image files in a ZIP using natural sort and renders the comic reader template.
func ListFilesInZip(w http.ResponseWriter, r *http.Request, zipName string, zipPath string) {
	rdr, err := zip.OpenReader(zipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rdr.Close()

	var files natural.StringSlice
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, f := range rdr.File {
		name := strings.ToLower(f.Name)
		for _, ext := range imageExts {
			if strings.HasSuffix(name, ext) {
				files = append(files, f.Name)
				break
			}
		}
	}
	sort.Sort(files)

	t, err := template.ParseFiles("web/templates/reader.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		ZipName string
		Files   []string
	}{
		ZipName: zipName,
		Files:   files,
	}
	t.Execute(w, data)
}

// ServeFileFromZip serves a file from the ZIP inline.
func ServeFileFromZip(w http.ResponseWriter, r *http.Request, zipName string, zipPath string, filePath string) {
	rdr, err := zip.OpenReader(zipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rdr.Close()

	for _, f := range rdr.File {
		if f.Name == filePath {
			rc, err := f.Open()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rc.Close()

			w.Header().Set("Content-Type", "image/jpeg") // Assume JPEG, can be dynamic
			w.Header().Set("Content-Disposition", `inline; filename="`+filePath+`"`)
			_, err = io.Copy(w, rc)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}
	}
	http.NotFound(w, r)
}
