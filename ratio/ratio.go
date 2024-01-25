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

type File struct {
	Code     int    `json:"code"`
	Comments int    `json:"comment"`
	Blanks   int    `json:"blank"`
	Path     string `json:"path"`
	Lang     string `json:"language"`
}

type Files []*File

type Ratio struct {
	Code      int   `json:"code"`
	Test      int   `json:"test"`
	CodeFiles Files `json:"code_files"`
	TestFiles Files `json:"test_files"`
}

type DiffRatio struct {
	A      float64 `json:"a"`
	B      float64 `json:"b"`
	Diff   float64 `json:"diff"`
	RatioA *Ratio  `json:"-"`
	RatioB *Ratio  `json:"-"`
}

func New() *Ratio {
	return &Ratio{}
}

func (r *Ratio) Compare(r2 *Ratio) *DiffRatio {
	d := &DiffRatio{
		RatioA: r,
		RatioB: r2,
	}
	var ratioA, ratioB float64
	if r != nil && r.Code != 0 {
		ratioA = float64(r.Test) / float64(r.Code)
	}
	if r2 != nil && r2.Code != 0 {
		ratioB = float64(r2.Test) / float64(r2.Code)
	}
	d.A = ratioA
	d.B = ratioB
	d.Diff = ratioA - ratioB
	return d
}

func (r *Ratio) DeleteFiles() {
	r.CodeFiles = Files{}
	r.TestFiles = Files{}
}

func Measure(root string, code, test []string) (*Ratio, error) {
	log.Printf("root: %s", root)
	ratio := New()
	defined := gocloc.NewDefinedLanguages()
	opts := gocloc.NewClocOptions()
	for i, p := range code {
		code[i] = filepath.FromSlash(p)
	}
	for i, p := range test {
		test[i] = filepath.FromSlash(p)
	}

	if err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
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

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

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
			match, err := doublestar.PathMatch(p, rel)
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
			match, err := doublestar.PathMatch(p, rel)
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
			if _, err := fmt.Fprintf(os.Stderr, "could not detect language: %s\n", path); err != nil {
				return err
			}
			return nil
		}
		l, ok := gocloc.Exts[ext]
		if !ok {
			if _, err := fmt.Fprintf(os.Stderr, "unsupported language (%s): %s\n", ext, path); err != nil {
				return err
			}
			return nil
		}
		cf := gocloc.AnalyzeFile(path, defined.Langs[l], opts)
		if isCode {
			log.Printf("code: %s,%d", rel, cf.Code)
			ratio.Code += int(cf.Code)
			ratio.CodeFiles = append(ratio.CodeFiles, &File{
				Code:     int(cf.Code),
				Comments: int(cf.Comments),
				Blanks:   int(cf.Blanks),
				Path:     rel,
				Lang:     cf.Lang,
			})
		}
		if isTest {
			log.Printf("test: %s,%d", rel, cf.Code)
			ratio.Test += int(cf.Code)
			ratio.TestFiles = append(ratio.TestFiles, &File{
				Code:     int(cf.Code),
				Comments: int(cf.Comments),
				Blanks:   int(cf.Blanks),
				Path:     rel,
				Lang:     cf.Lang,
			})
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
