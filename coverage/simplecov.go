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
	Lines []any `json:"lines"`
}

func NewSimplecov() *Simplecov {
	return &Simplecov{}
}

func (s *Simplecov) Name() string {
	return "SimpleCov"
}

type skipLine map[int]struct{}

type skipLines map[string]skipLine

func (sl skipLines) add(path string, line int) {
	if _, ok := sl[path]; !ok {
		sl[path] = skipLine{}
	}
	sl[path][line] = struct{}{}
}

func (sl skipLines) exists(path string, line int) bool {
	if _, ok := sl[path]; !ok {
		return false
	}
	if _, ok := sl[path][line]; !ok {
		return false
	}
	return true
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
	fcovs := map[string]*FileCoverage{}
	sls := skipLines{}
	for _, c := range r {
		for fn, fc := range c.Coverage {
			fcov, ok := fcovs[fn]
			if !ok {
				fcov = NewFileCoverage(fn, TypeLOC)
				fcovs[fn] = fcov
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
				case nil:
					sls.add(fn, ll)
				}
			}
		}
	}

	cov.Total = 0
	cov.Covered = 0
	for _, fcov := range cov.Files {
		blocks := fcov.Blocks
		fcov.Blocks = BlockCoverages{}
		for _, b := range blocks {
			if *b.Count == 0 && sls.exists(fcov.File, *b.StartLine) {
				continue
			}
			fcov.Blocks = append(fcov.Blocks, b)
		}

		lcs := fcov.Blocks.ToLineCoverages()
		fcov.Total = lcs.Total()
		fcov.Covered = lcs.Covered()
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
		Coverage map[string]any `json:"coverage"`
	}{}
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	c.Coverage = map[string]SimplecovFileCoverage{}
	for k, l := range s.Coverage {
		switch v := l.(type) {
		case map[string]any:
			c.Coverage[k] = SimplecovFileCoverage{
				Lines: v["lines"].([]any),
			}
		case []any:
			c.Coverage[k] = SimplecovFileCoverage{
				Lines: v,
			}
		default:
			return errors.New("unsupported SimpleCov report format")
		}
	}
	return nil
}
