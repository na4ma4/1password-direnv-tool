package model

import (
	"path/filepath"
	"sort"
)

type FileList struct {
	root  string
	files []string
}

func NewFileList(root string) *FileList {
	return &FileList{
		root:  root,
		files: []string{},
	}
}

func (f *FileList) Len() int           { return len(f.files) }
func (f *FileList) Less(i, j int) bool { return f.files[i] < f.files[j] }
func (f *FileList) Swap(i, j int)      { f.files[i], f.files[j] = f.files[j], f.files[i] }
func (f *FileList) Append(files ...string) *FileList {
	f.files = append(f.files, files...)
	return f
}

func (f *FileList) Merge(t *FileList) *FileList {
	if t == nil {
		return f
	}

	if f == nil {
		return t
	}

	if f.root == "" {
		f.root = t.root
	}

	f.files = append(f.files, t.files...)
	return f
}

func (f *FileList) GetFiles() []string {
	if f == nil {
		return nil
	}

	o := make(map[string]any, len(f.files))
	for _, file := range f.files {
		o[filepath.Join(f.root, file)] = nil
	}

	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

// func (f *FileList) Strings() []string {
// 	return f.files
// }
