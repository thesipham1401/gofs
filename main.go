package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

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
		for _, e := range dir {
			fmt.Fprintln(w, e.Name())
		}
	} else {
		http.ServeFile(w, r, path)
	}
}

func main() {
	http.HandleFunc("/{path...}", servePath)
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
