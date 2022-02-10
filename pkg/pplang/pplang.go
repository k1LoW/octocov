package pplang

import (
	"errors"
	"io/fs"
	"os"
)

func Detect(dir string) (string, error) {
	return DetectFS(os.DirFS(dir))
}

func DetectFS(fsys fs.FS) (string, error) {
	if fi, err := fs.Stat(fsys, "go.mod"); err == nil && !fi.IsDir() {
		return "Go", nil
	}
	return "", errors.New("can not detect programming language")
}
