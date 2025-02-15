package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ISize interface {
	SizeStr() string
}

type FileSize int64
type DirSize uint64

func (s FileSize) SizeStr() string {
	switch {
	case s < 1024:
		return fmt.Sprintf("%v bytes", s)
	case s >= 1024 && s < 1024*1024:
		return fmt.Sprintf("%0.2f KB", float64(s)/1024)
	case s >= 1024*1024 && s < 1024*1024*1024:
		return fmt.Sprintf("%0.2f MB", float64(s)/(1024*1024))
	default:
		return fmt.Sprintf("%0.2f GB", float64(s)/(1024*1024*1024))
	}
}

func (s DirSize) SizeStr() string {
	return fmt.Sprintf("%v items", s)
}

type Item struct {
	Name         string
	LastModified time.Time
	Size         ISize
	IsDir        bool
}

type Model struct {
	Path  string
	Items []Item
}

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
		items := make([]Item, 0)
		for _, e := range dir {
			info, _ := e.Info()
			if e.IsDir() {
				d, _ := os.ReadDir(e.Name())
				items = append(items, Item{IsDir: true, Name: e.Name(), LastModified: info.ModTime(), Size: DirSize(len(d))})
			} else {
				items = append(items, Item{IsDir: false, Name: e.Name(), LastModified: info.ModTime(), Size: FileSize(info.Size())})
			}
		}
		tmpl.Execute(w, Model{Path: path, Items: items})
	} else {
		http.ServeFile(w, r, path)
	}
}

var tmpl *template.Template

func main() {
	tmpl = template.Must(template.ParseFiles("template.html"))
	http.HandleFunc("/{path...}", servePath)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
