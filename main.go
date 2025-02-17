package main

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"strings"

	"github.com/ndtoan96/gofs/model"
	"github.com/spf13/pflag"
)

var tmpl map[string]*template.Template
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
		if err := tmpl["files"].Execute(w, model.FilesPageModel{Path: model.Path(path), Items: items, AllowWrite: allowWrite}); err != nil {
			log.Fatalln("[ERROR]", err)
		}

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
	var err error = nil
	if r.FormValue("__gofs-delete") == "Delete" {
		if len(names) > 0 {
			err = tmpl["delete"].Execute(w, model.DeletePageModel{Path: model.Path(currentDir), Names: names})
		} else {
			http.Redirect(w, r, currentDir, http.StatusMovedPermanently)
		}
	} else if r.FormValue("__gofs-archive") == "Archive" {
		if len(names) > 0 {
			err = tmpl["archive"].Execute(w, model.ArchivePageModel{Path: model.Path(currentDir), Names: names})
		} else {
			http.Redirect(w, r, currentDir, http.StatusMovedPermanently)
		}
	} else if r.FormValue("__gofs-new-folder") == "New Folder" {
		err = tmpl["new-folder"].Execute(w, model.NewFolderPageModel{Path: model.Path(currentDir)})
	} else if r.FormValue("__gofs-upload") == "Upload Files" {
		err = tmpl["upload"].Execute(w, model.NewFolderPageModel{Path: model.Path(currentDir)})
	} else if r.FormValue("__gofs-rename") == "Rename" {
		if len(names) > 0 {
			err = tmpl["rename"].Execute(w, model.RenamePageModel{Path: model.Path(currentDir), OldNames: names})
		} else {
			http.Redirect(w, r, currentDir, http.StatusMovedPermanently)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	if err != nil {
		log.Fatalln("[ERROR]", err)
	}
}

func delete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	p := r.FormValue("path")
	if r.FormValue("submit") == "Yes" {
		items := r.PostForm["items"]
		for _, item := range items {
			err := os.RemoveAll(path.Join(p, item))
			if err != nil {
				log.Println("[ERROR]", err)
			}
		}
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func newFolder(w http.ResponseWriter, r *http.Request) {
	p := r.FormValue("path")
	name := r.FormValue("name")
	if r.FormValue("submit") == "Create" {
		os.Mkdir(path.Join(p, name), 0666)
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func archive(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	p := r.FormValue("path")
	name := r.FormValue("name")
	name += ".zip"
	if r.FormValue("submit") == "Archive" {
		items := r.PostForm["items"]
		err := zipFilesAndFolders(p, name, items)
		if err != nil {
			log.Println("[ERROR]", err)
		}
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func zipFilesAndFolders(dir string, output string, items []string) error {
	archive, err := os.Create(path.Join(dir, output))
	if err != nil {
		return err
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()
	for len(items) > 0 {
		item := items[0]
		items = items[1:]
		item_path := path.Join(dir, item)
		info, err := os.Stat(item_path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			dirEntry, err := os.ReadDir(item_path)
			if err != nil {
				return err
			}
			if len(dirEntry) == 0 {
				if _, err := zipWriter.Create(item + "/"); err != nil {
					return err
				}
			} else {
				for _, sub := range dirEntry {
					items = append(items, path.Join(item, sub.Name()))
				}
			}

		} else {
			f, err := os.Open(item_path)
			if err != nil {
				return err
			}
			defer f.Close()
			w, err := zipWriter.Create(item)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, f); err != nil {
				return err
			}
		}
	}
	return nil
}

func rename(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	p := r.FormValue("path")
	oldNames := make([]string, 0)
	newNames := make([]string, 0)
	for k, v := range r.PostForm {
		if strings.HasPrefix(k, "oldname-") {
			oldNames = append(oldNames, strings.TrimPrefix(k, "oldname-"))
			newNames = append(newNames, v[0])
		}
	}
	if r.FormValue("submit") == "Rename" {
		for i, oldName := range oldNames {
			newName := newNames[i]
			if err := os.Rename(path.Join(p, oldName), path.Join(p, newName)); err != nil {
				log.Println("[ERROR]", err)
			}
		}
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func main() {
	pflag.BoolVarP(&allowWrite, "write", "w", false, "Allow write access")
	pflag.IntVarP(&port, "port", "p", 8080, "Port to listen")
	pflag.Parse()

	tmpl = make(map[string]*template.Template)
	tmpl["delete"] = template.Must(template.ParseFiles("templates/layout.html", "templates/delete.html"))
	tmpl["new-folder"] = template.Must(template.ParseFiles("templates/layout.html", "templates/new-folder.html"))
	tmpl["archive"] = template.Must(template.ParseFiles("templates/layout.html", "templates/archive.html"))
	tmpl["files"] = template.Must(template.ParseFiles("templates/layout.html", "templates/files.html"))
	tmpl["upload"] = template.Must(template.ParseFiles("templates/layout.html", "templates/upload.html"))
	tmpl["rename"] = template.Must(template.ParseFiles("templates/layout.html", "templates/rename.html"))

	http.HandleFunc("GET /{path...}", servePath)
	http.HandleFunc("POST /confirm", confirmAction)
	http.HandleFunc("POST /delete", delete)
	http.HandleFunc("POST /new-folder", newFolder)
	http.HandleFunc("POST /archive", archive)
	http.HandleFunc("POST /rename", rename)
	http.HandleFunc("GET /__gofs__/style.css", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "style.css") })
	log.Printf("Starting server at localhost:%v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%v", port), nil))
}
