package coverage

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/goccy/go-json"
)

var _ Processor = (*Simplecov)(nil)

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

func (s *Simplecov) Name() string {
	return "SimpleCov"
}

func (s *Simplecov) ParseReport(path string) (*Coverage, string, error) {
	rp, err := s.detectReportPath(path)
	if err != nil {
		return nil, "", err
	}
	b, err := ioutil.ReadFile(filepath.Clean(rp))
	if err != nil {
		return nil, "", err
	}
	r := SimplecovReport{}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, "", err
	}
	cov := New()
	cov.Type = TypeLOC
	cov.Format = s.Name()
	for _, c := range r {
		for fn, fc := range c.Coverage {
			fcov := NewFileCoverage(fn)
			for l, c := range fc.Lines {
				ll := l + 1
				switch v := c.(type) {
				case float64:
					fcov.Total += 1
					count := int(v)
					if count > 0 {
						fcov.Covered += 1
					}

					fcov.Blocks = append(fcov.Blocks, &BlockCoverage{
						Type:      TypeLOC,
						StartLine: &ll,
						EndLine:   &ll,
						Count:     &count,
					})
				}
			}
			cov.Total += fcov.Total
			cov.Covered += fcov.Covered
			cov.Files = append(cov.Files, fcov)
		}
	}
	return cov, rp, nil
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
