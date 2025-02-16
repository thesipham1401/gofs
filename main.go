package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"

	"strings"

	"github.com/ndtoan96/gofs/model"
	"github.com/spf13/pflag"
)

var tmpl map[string]*template.Template
var delTmpl *template.Template
var allowWrite bool
var port int

func servePath(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if strings.HasPrefix(path, "..") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if path == "" {
		path = "."
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if info.IsDir() {
		dir, err := os.ReadDir(path)
		if err != nil {
			log.Println("[ERROR]", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Serve directory
		items := make([]model.Item, 0)
		for _, e := range dir {
			info, _ := e.Info()
			if e.IsDir() {
				d, _ := os.ReadDir(e.Name())
				items = append(items, model.Item{IsDir: true, Name: e.Name(), LastModified: info.ModTime(), Size: model.DirSize(len(d))})
			} else {
				items = append(items, model.Item{IsDir: false, Name: e.Name(), LastModified: info.ModTime(), Size: model.FileSize(info.Size())})
			}
		}
		tmpl["files"].Execute(w, model.FilesPageModel{Path: model.Path(path), Items: items, AllowWrite: allowWrite})
	} else {
		http.ServeFile(w, r, path)
	}
}

func confirmAction(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	names := make([]string, 0)
	for k, v := range r.PostForm {
		if v[0] == "on" {
			names = append(names, k)
		}
	}
	currentDir := r.FormValue("path")
	if r.FormValue("__gofs-delete") == "Delete" {
		delTmpl.Execute(w, model.DeletePageModel{Path: model.Path(currentDir), Names: names})
	} else if r.FormValue("__gofs-archive") == "Archive" {
		tmpl["archive"].Execute(w, model.ArchivePageModel{Path: model.Path(currentDir), Names: names})
	} else if r.FormValue("__gofs-new-folder") == "New Folder" {
		tmpl["new-folder"].Execute(w, model.NewFolderPageModel{Path: model.Path(currentDir)})
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func delete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	p := r.FormValue("path")
	items := r.PostForm["items"]
	for _, item := range items {
		err := os.RemoveAll(path.Join(p, item))
		if err != nil {
			log.Println("[ERROR]", err)
		}
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func newFolder(w http.ResponseWriter, r *http.Request) {

}

func archive(w http.ResponseWriter, r *http.Request) {

}

func main() {
	pflag.BoolVarP(&allowWrite, "write", "w", false, "Allow write access")
	pflag.IntVarP(&port, "port", "p", 8080, "Port to listen")
	pflag.Parse()

	tmpl = make(map[string]*template.Template)
	delTmpl = template.Must(template.ParseFiles("templates/layout.html", "templates/delete.html"))
	tmpl["new-folder"] = template.Must(template.ParseFiles("templates/layout.html", "templates/new-folder.html"))
	tmpl["archive"] = template.Must(template.ParseFiles("templates/layout.html", "templates/archive.html"))
	tmpl["files"] = template.Must(template.ParseFiles("templates/layout.html", "templates/files.html"))

	http.HandleFunc("GET /{path...}", servePath)
	http.HandleFunc("POST /confirm", confirmAction)
	http.HandleFunc("POST /delete", delete)
	http.HandleFunc("POST /new_folder", newFolder)
	http.HandleFunc("POST /archive", archive)
	http.HandleFunc("GET /__gofs__/style.css", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "style.css") })
	log.Printf("Starting server at localhost:%v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%v", port), nil))
}
