package coverage

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/goccy/go-json"
)

var _ Processor = (*Simplecov)(nil)

var SimplecovDefaultPath = []string{"coverage", ".resultset.json"}

type Simplecov struct{}

type SimplecovReport map[string]SimplecovCoverage

type SimplecovCoverage struct {
	Coverage map[string]SimplecovFileCoverage `json:"coverage"`
}

type SimplecovFileCoverage struct {
	Lines []interface{} `json:"lines"`
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
	b, err := os.ReadFile(filepath.Clean(rp))
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
			var fcov *FileCoverage
			fcov, err = cov.Files.FindByFile(fn)
			if err != nil {
				fcov = NewFileCoverage(fn)
				cov.Files = append(cov.Files, fcov)
			}
			for l, c := range fc.Lines {
				ll := l + 1
				switch v := c.(type) {
				case float64:
					count := int(v)
					fcov.Blocks = append(fcov.Blocks, &BlockCoverage{
						Type:      TypeLOC,
						StartLine: &ll,
						EndLine:   &ll,
						Count:     &count,
					})
				}
			}
			lcs := fcov.Blocks.ToLineCoverages()
			fcov.Total = lcs.Total()
			fcov.Covered = lcs.Covered()
		}
	}

	cov.Total = 0
	cov.Covered = 0
	for _, fcov := range cov.Files {
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
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

func (c *SimplecovCoverage) UnmarshalJSON(data []byte) error {
	s := struct {
		Coverage map[string]interface{} `json:"coverage"`
	}{}
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	c.Coverage = map[string]SimplecovFileCoverage{}
	for k, l := range s.Coverage {
		switch v := l.(type) {
		case map[string]interface{}:
			c.Coverage[k] = SimplecovFileCoverage{
				Lines: v["lines"].([]interface{}),
			}
		case []interface{}:
			c.Coverage[k] = SimplecovFileCoverage{
				Lines: v,
			}
		default:
			return errors.New("unsupported SimpleCov report format")
		}
	}
	return nil
}
