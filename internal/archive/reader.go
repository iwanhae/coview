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
#controls { position: fixed; bottom: 10px; left: 50%; transform: translateX(-50%); text-align: center; z-index: 10; cursor: pointer; padding: 10px; background: rgba(0,0,0,0.5); border-radius: 10px; }
#controls button { padding: 10px 20px; margin: 0 10px; font-size: 16px; border: none; background: #333; color: white; border-radius: 5px; cursor: pointer; }
#pageInfo { margin: 0 10px; }
#modal {
  display: none;
  position: fixed;
  z-index: 20;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0,0,0,0.8);
}
#modal-content {
  background-color: #333;
  margin: 15% auto;
  padding: 20px;
  border-radius: 10px;
  width: 80%;
  max-width: 400px;
  text-align: center;
  color: white;
}
#modal input { width: 100px; margin: 10px; padding: 5px; }
#modal button { padding: 10px 15px; margin: 5px; font-size: 16px; border: none; background: #555; color: white; border-radius: 5px; cursor: pointer; }
@media (max-width: 600px) {
  #controls { bottom: 5px; padding: 5px; }
  #controls button { padding: 8px 16px; font-size: 14px; margin: 0 5px; }
  #pageInfo { font-size: 12px; }
  #modal-content { margin: 20% auto; width: 90%; padding: 15px; }
  #modal button { padding: 8px 12px; font-size: 14px; }
}
</style>
<script>
let currentPage = 0;
const pages = [{{range .Files}}"{{.}}",{{end}}];
const totalPages = pages.length;
function loadPage(index) {
  if (index < 0 || index >= totalPages) return;
  currentPage = index;
  document.getElementById('image').src = '/' + "{{.ZipName}}" + '/' + pages[index] + '?t=' + Date.now();
  document.getElementById('pageInfo').textContent = (currentPage + 1) + ' / ' + totalPages;
  document.getElementById('pageInput').value = currentPage + 1;
}
function nextPage() { loadPage(currentPage + 1); }
function prevPage() { loadPage(currentPage - 1); }
function goToPage() {
  const pageNum = parseInt(document.getElementById('pageInput').value) - 1;
  if (!isNaN(pageNum) && pageNum >= 0 && pageNum < totalPages) {
    loadPage(pageNum);
  }
}
function toggleModal() {
  const modal = document.getElementById('modal');
  modal.style.display = modal.style.display === 'block' ? 'none' : 'block';
  if (modal.style.display === 'block') {
    document.getElementById('pageInput').value = currentPage + 1;
  }
}
function closeModal(event) {
  if (event.target.id === 'modal') {
    document.getElementById('modal').style.display = 'none';
  }
}
document.addEventListener('keydown', (e) => {
  if (e.key === 'ArrowRight' || e.key === ' ') nextPage();
  if (e.key === 'ArrowLeft') prevPage();
  if (e.key === 'Escape') closeModal();
});
document.addEventListener('touchend', (e) => {
  if (e.target.id === 'viewer') {
    nextPage();
  }
});
window.onload = () => {
  loadPage(0);
};
</script>
</head>
<body>
<div id="viewer" onclick="nextPage()">
  <img id="image" src="" alt="Comic Page">
</div>
<div id="controls" onclick="toggleModal()">
  <span id="pageInfo"></span>
</div>
<div id="modal" onclick="closeModal(event)">
  <div id="modal-content" onclick="event.stopPropagation();">
    <h3>Navigation</h3>
    <button onclick="prevPage(); closeModal();">Previous</button>
    <button onclick="nextPage(); closeModal();">Next</button>
    <br>
    Go to page: <input type="number" id="pageInput" min="1" max="{{len .Files}}" value="1">
    <button onclick="goToPage(); closeModal();">Go</button>
    <br>
    <button onclick="closeModal();">Close</button>
  </div>
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
