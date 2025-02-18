package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"strings"

	"github.com/ndtoan96/gofs/model"
	cp "github.com/otiai10/copy"
	"github.com/spf13/pflag"
)

var tmpl map[string]*template.Template
var allowWrite bool
var port int

func servePath(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	selectState := r.URL.Query().Get("select")
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
		if err := tmpl["files"].Execute(w, model.FilesPageModel{Path: model.Path(path), Items: items, AllowWrite: allowWrite, SelectState: selectState}); err != nil {
			log.Fatalln("[ERROR]", err)
		}

	} else {
		http.ServeFile(w, r, path)
	}
}

func action(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	names := r.PostForm["select"]
	p := r.FormValue("path")
	var err error = nil
	action := r.FormValue("action")
	switch action {
	case "delete":
		if len(names) > 0 {
			err = tmpl["delete"].Execute(w, model.DeletePageModel{Path: model.Path(p), Names: names})
		} else {
			http.Redirect(w, r, p, http.StatusMovedPermanently)
		}
	case "archive":
		if len(names) > 0 {
			err = tmpl["archive"].Execute(w, model.ArchivePageModel{Path: model.Path(p), Names: names})
		} else {
			http.Redirect(w, r, p, http.StatusMovedPermanently)
		}
	case "new-folder":
		err = tmpl["new-folder"].Execute(w, model.NewFolderPageModel{Path: model.Path(p)})
	case "upload":
		err = tmpl["upload"].Execute(w, model.NewFolderPageModel{Path: model.Path(p)})
	case "rename":
		if len(names) > 0 {
			err = tmpl["rename"].Execute(w, model.RenamePageModel{Path: model.Path(p), OldNames: names})
		} else {
			http.Redirect(w, r, p, http.StatusMovedPermanently)
		}
	case "download":
		if len(names) == 0 {
			http.Redirect(w, r, p, http.StatusMovedPermanently)
		} else if len(names) == 1 {
			item := names[0]
			itemPath := path.Join(p, item)
			var stat os.FileInfo
			stat, err = os.Stat(itemPath)
			if err != nil {
				break
			}
			if stat.IsDir() {
				var buf bytes.Buffer
				err = zipFilesAndFolders(&buf, p, names)
				if err != nil {
					break
				}
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v.zip", item))
				w.Header().Set("Content-Type", "application/zip")
				io.Copy(w, &buf)
			} else {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v", item))
				http.ServeFile(w, r, itemPath)
			}
		} else {
			var buf bytes.Buffer
			err = zipFilesAndFolders(&buf, p, names)
			if err != nil {
				break
			}
			zipName := path.Base(p)
			if zipName == "" || zipName == "." {
				zipName = "download"
			}
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v.zip", zipName))
			w.Header().Set("Content-Type", "application/zip")
			io.Copy(w, &buf)
		}
	case "copy":
		if len(names) > 0 {
			http.SetCookie(w, &http.Cookie{Name: "cpdir", Value: p})
			encodedNames := strings.Join(names, "!$!")
			http.SetCookie(w, &http.Cookie{Name: "cpitems", Value: string(encodedNames)})
			http.Redirect(w, r, r.Referer(), http.StatusMovedPermanently)
		}
	case "paste":
		var cpdir *http.Cookie
		cpdir, err = r.Cookie("cpdir")
		if err != nil {
			http.Redirect(w, r, r.Referer(), http.StatusMovedPermanently)
			return
		}
		var encodedNames *http.Cookie
		encodedNames, err = r.Cookie("cpitems")
		if err != nil {
			http.Redirect(w, r, r.Referer(), http.StatusMovedPermanently)
			return
		}
		decodedNames := strings.Split(encodedNames.Value, "!$!")
		for _, name := range decodedNames {
			srcPath := path.Join(cpdir.Value, name)
			destPath := path.Join(p, name)
			_, err = os.Stat(destPath)
			for !os.IsNotExist(err) {
				ext := path.Ext(destPath)
				destPath = path.Join(p, strings.TrimSuffix(path.Base(destPath), ext)+"_copy"+ext)
				_, err = os.Stat(destPath)
			}
			err = nil
			// Cannot copy a folder into itself
			if !strings.HasPrefix(destPath, srcPath) {
				cp.Copy(srcPath, destPath)
			}
		}
		http.Redirect(w, r, r.Referer(), http.StatusMovedPermanently)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
	if err != nil {
		log.Println("[ERROR]", err)
		w.WriteHeader(http.StatusInternalServerError)
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
		archive, err := os.Create(path.Join(p, name))
		if err != nil {
			log.Println("[ERROR]", err)
		}
		defer archive.Close()
		err = zipFilesAndFolders(archive, p, items)
		if err != nil {
			log.Println("[ERROR]", err)
		}
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func zipFilesAndFolders(writer io.Writer, dir string, items []string) error {
	zipWriter := zip.NewWriter(writer)
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
			oldNamePath := path.Join(p, oldName)
			newNamePath := path.Join(p, newName)
			log.Printf("[INFO] Rename `%v` -> `%v`", oldNamePath, newNamePath)
			if err := os.Rename(oldNamePath, newNamePath); err != nil {
				log.Println("[ERROR]", err)
			}
		}
	}
	http.Redirect(w, r, p, http.StatusMovedPermanently)
}

func upload(w http.ResponseWriter, r *http.Request) {
	p := r.FormValue("path")
	if r.FormValue("submit") == "Upload" {
		for _, f := range r.MultipartForm.File["files"] {
			filePath := path.Join(p, f.Filename)
			log.Printf("[INFO] Upload `%v`\n", filePath)
			w, err := os.Create(filePath)
			if err != nil {
				log.Println("[ERROR]", err)
			}
			defer w.Close()
			r, err := f.Open()
			if err != nil {
				log.Println("[ERROR]", err)
			}
			defer r.Close()
			_, err = io.Copy(w, r)
			if err != nil {
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
	http.HandleFunc("POST /action", action)
	http.HandleFunc("POST /delete", delete)
	http.HandleFunc("POST /new-folder", newFolder)
	http.HandleFunc("POST /archive", archive)
	http.HandleFunc("POST /rename", rename)
	http.HandleFunc("POST /upload", upload)
	http.HandleFunc("GET /__gofs__/style.css", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "style.css") })
	log.Printf("Starting server at localhost:%v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("localhost:%v", port), nil))
}
