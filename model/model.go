package model

import (
	"fmt"
	"path"
	"slices"
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
		return fmt.Sprintf("%v B", s)
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

type ParentItem struct {
	Name string
	Path string
}

func (model Model) Parents() []ParentItem {
	items := make([]ParentItem, 0)
	for parent := path.Dir(model.Path); parent != "."; parent = path.Dir(parent) {
		items = append(items, ParentItem{Path: parent, Name: path.Base(parent)})
	}
	slices.Reverse(items)
	return items
}

func (model Model) CurrentDir() string {
	return path.Base(model.Path)
}
