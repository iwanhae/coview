package archive

import (
	"archive/zip"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iwanhae/coview/internal/config"
	"github.com/iwanhae/coview/pkg/natural"
)

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

	var zipNames []string
	for _, f := range files {
		zipNames = append(zipNames, filepath.Base(f))
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
<title>Comic ZIP Files</title>
</head>
<body>
<h1>Comic ZIP Files</h1>

<h2>Upload New ZIP</h2>
<form action="/upload" method="post" enctype="multipart/form-data">
  <input type="file" name="file" accept=".zip" required>
  <button type="submit">Upload</button>
</form>

<h2>Available Comics</h2>
<ul>
{{range .}}
<li><a href="/{{.}}">View {{.}}</a></li>
{{end}}
</ul>
</body>
</html>
`
	t := template.Must(template.New("list").Parse(tmpl))
	t.Execute(w, zipNames)
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
<title>{{.ZipName}} - Comic Reader</title>
<style>
body { margin: 0; padding: 0; background: #000; color: white; font-family: Arial; }
#viewer { width: 100vw; height: 100vh; display: flex; justify-content: center; align-items: center; }
#image { max-width: 100%; max-height: 100%; object-fit: contain; }
#controls { position: fixed; bottom: 10px; left: 50%; transform: translateX(-50%); text-align: center; }
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
