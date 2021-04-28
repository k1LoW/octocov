package coverage

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/goccy/go-json"
)

var SimplecovDefaultPath = []string{"coverage", ".resultset.json"}

type Simplecov struct{}

type SimplecovReport map[string]SimplecovCoverage

type SimplecovCoverage struct {
	Coverage map[string]SimplecovFileCoverage
}

type SimplecovFileCoverage struct {
	Lines []interface{}
}

func NewSimplecov() *Simplecov {
	return &Simplecov{}
}

func (s *Simplecov) ParseReport(path string) (*Coverage, error) {
	path, err := s.detectReportPath(path)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	r := SimplecovReport{}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	cov := New()
	cov.Type = TypeLOC
	cov.Format = "SimpleCov"
	for _, c := range r {
		for fn, fc := range c.Coverage {
			fcov := NewFileCoverage(fn)
			for _, l := range fc.Lines {
				switch v := l.(type) {
				case float64:
					fcov.Total += 1
					if v > 0 {
						fcov.Covered += 1
					}
				}
			}
			cov.Total += fcov.Total
			cov.Covered += fcov.Covered
			cov.Files = append(cov.Files, fcov)
		}
	}
	return cov, nil
}

func (s *Simplecov) detectReportPath(path string) (string, error) {
	p, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if p.IsDir() {
		// path/to/coverage/.resultset.json
		np := filepath.Join(path, SimplecovDefaultPath[0], SimplecovDefaultPath[1])
		if _, err := os.Stat(np); err != nil {
			// path/to/.resultset.json
			np = filepath.Join(path, SimplecovDefaultPath[1])
			if _, err := os.Stat(np); err != nil {
				return "", err
			}
		}
		path = np
	}
	return path, nil
}
