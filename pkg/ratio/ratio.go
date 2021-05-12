package ratio

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/hhatto/gocloc"
)

type Ratio struct {
	Code      int      `json:"code"`
	Test      int      `json:"test"`
	CodeFiles []string `json:"-"`
	TestFiles []string `json:"-"`
}

func New() *Ratio {
	return &Ratio{}
}

func Measure(root string, code, test []string) (*Ratio, error) {
	ratio := New()
	defined := gocloc.NewDefinedLanguages()
	opts := gocloc.NewClocOptions()

	if err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() {
			if ignore(path) {
				return filepath.SkipDir
			}
			return nil
		}
		if ignore(path) {
			return nil
		}

		isCode := false
		isTest := false

		// check path
		if len(code) == 0 {
			isCode = true
		}
		for _, p := range code {
			not := false
			if strings.HasPrefix(p, "!") {
				p = strings.TrimPrefix(p, "!")
				not = true
			}
			match, err := doublestar.PathMatch(p, path)
			if err != nil {
				return err
			}
			if match {
				if not {
					isCode = false
				} else {
					isCode = true
				}
			}
		}
		// test
		for _, p := range test {
			not := false
			if strings.HasPrefix(p, "!") {
				p = strings.TrimPrefix(p, "!")
				not = true
			}
			match, err := doublestar.PathMatch(p, path)
			if err != nil {
				return err
			}
			if match {
				if not {
					isTest = false
				} else {
					isTest = true
				}
			}
		}
		if !isCode && !isTest {
			return nil
		}
		ext, ok := getFileType(path)
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "could not detect language: %s\n", path)
			return nil
		}
		l, ok := gocloc.Exts[ext]
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "unsupported language: %s\n", ext)
			return nil
		}
		cf := gocloc.AnalyzeFile(path, defined.Langs[l], opts)
		if isCode {
			log.Printf("code: %s,%d", path, cf.Code)
			ratio.Code += int(cf.Code)
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			ratio.CodeFiles = append(ratio.CodeFiles, rel)
		}
		if isTest {
			log.Printf("test: %s,%d", path, cf.Code)
			ratio.Test += int(cf.Code)
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			ratio.TestFiles = append(ratio.TestFiles, rel)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if ratio.Code == 0 {
		return nil, fmt.Errorf("could not count code: %s", code)
	}
	return ratio, nil
}

var ignores = []string{
	".bzr", ".cvs", ".hg", ".git", ".svn",
	".github", ".gitignore", ".gitkeep",
}

func ignore(path string) bool {
	return contains(ignores, filepath.Base(path))
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}
