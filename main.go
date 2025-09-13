package main

import (
	"archive/zip"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/iwanhae/coview/natural"
)

var dataDir = "data"

func main() {
	http.HandleFunc("/", handler)

	log.Println("Server starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		listZips(w, r)
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

	zipPath := filepath.Join(dataDir, zipName)

	if len(parts) == 1 {
		listFilesInZip(w, r, zipName, zipPath)
		return
	}

	if len(parts) >= 2 {
		filePath := strings.Join(parts[1:], "/")
		serveFileFromZip(w, r, zipName, zipPath, filePath)
		return
	}

	http.NotFound(w, r)
}

func listZips(w http.ResponseWriter, r *http.Request) {
	files, err := filepath.Glob(filepath.Join(dataDir, "*.zip"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var zipNames []string
	for _, f := range files {
		zipNames = append(zipNames, filepath.Base(f))
	}

	tmpl := `
<html>
<head><title>Zip Files</title></head>
<body>
<h1>Zip Files in data</h1>
<ul>
{{range .}}
<li><a href="/{{.}}">{{.}}</a></li>
{{end}}
</ul>
</body>
</html>
`
	t := template.Must(template.New("list").Parse(tmpl))
	t.Execute(w, zipNames)
}

func listFilesInZip(w http.ResponseWriter, r *http.Request, zipName string, zipPath string) {
	rdr, err := zip.OpenReader(zipPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rdr.Close()

	var files natural.StringSlice
	for _, f := range rdr.File {
		files = append(files, f.Name)
	}
	sort.Sort(files)

	tmpl := `
<html>
<head><title>{{.ZipName}} Contents</title></head>
<body>
<h1>Contents of {{.ZipName}}</h1>
<ul>
{{range .Files}}
<li><a href="/{{$.ZipName}}/{{.}}">{{.}}</a></li>
{{end}}
</ul>
<a href="/">Back to list</a>
</body>
</html>
`
	t := template.Must(template.New("zip").Parse(tmpl))
	data := struct {
		ZipName string
		Files   []string
	}{
		ZipName: zipName,
		Files:   files,
	}
	t.Execute(w, data)
}

func serveFileFromZip(w http.ResponseWriter, r *http.Request, zipName string, zipPath string, filePath string) {
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

			w.Header().Set("Content-Disposition", `inline; filename="`+filePath+`"`)
			io.Copy(w, rc)
			return
		}
	}
	http.NotFound(w, r)
}
