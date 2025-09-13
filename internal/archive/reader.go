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

	tmpl := `
<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.ZipName}} - Comic Reader</title>
<style>
body { margin: 0; padding: 0; background: #000; color: white; font-family: Arial; }
#viewer { width: 100vw; height: 100vh; display: flex; justify-content: center; align-items: center; }
#image { max-width: 100%; max-height: 100%; object-fit: contain; }
#controls { position: fixed; bottom: 10px; left: 50%; transform: translateX(-50%); text-align: center; z-index: 10; }
#controls button { padding: 10px 20px; margin: 0 10px; font-size: 16px; border: none; background: #333; color: white; border-radius: 5px; cursor: pointer; }
#pageInfo { margin: 0 10px; }
@media (max-width: 600px) {
  #controls { bottom: 5px; }
  #controls button { padding: 8px 16px; font-size: 14px; margin: 0 5px; }
  #pageInfo { font-size: 12px; }
}
</style>
<script>
let currentPage = 0;
const pages = [{{range .Files}}"{{.}}",{{end}}];
function loadPage(index) {
  if (index < 0 || index >= pages.length) return;
  currentPage = index;
  document.getElementById('image').src = '/' + "{{.ZipName}}" + '/' + pages[index] + '?t=' + Date.now();
  document.getElementById('pageInfo').textContent = (currentPage + 1) + ' / ' + pages.length;
}
function nextPage() { loadPage(currentPage + 1); }
function prevPage() { loadPage(currentPage - 1); }
document.addEventListener('keydown', (e) => {
  if (e.key === 'ArrowRight' || e.key === ' ') nextPage();
  if (e.key === 'ArrowLeft') prevPage();
});
document.addEventListener('touchend', (e) => {
  const touch = e.changedTouches[0];
  const x = (touch.clientX / window.innerWidth) * 100;
  if (x > 50) nextPage();
  else prevPage();
});
window.onload = () => {
  loadPage(0);
};
</script>
</head>
<body>
<div id="viewer">
  <img id="image" src="" alt="Comic Page">
</div>
<div id="controls">
  <button onclick="prevPage()">Previous</button>
  <span id="pageInfo"></span>
  <button onclick="nextPage()">Next</button>
</div>
</body>
</html>
`
	t := template.Must(template.New("reader").Parse(tmpl))
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
