package coverage

import (
	"fmt"
	"strings"
)

type Type string

const (
	TypeLOC  Type = "loc"
	TypeStmt Type = "statement"
)

type Coverage struct {
	Type    Type          `json:"type"`
	Format  string        `json:"format"`
	Total   int           `json:"total"`
	Covered int           `json:"covered"`
	Files   FileCoverages `json:"files"`
}

type FileCoverage struct {
	File    string         `json:"file"`
	Total   int            `json:"total"`
	Covered int            `json:"covered"`
	Blocks  BlockCoverages `json:"blocks"`
	cache   map[int]BlockCoverages
}

type FileCoverages []*FileCoverage

type BlockCoverage struct {
	Type      Type `json:"type"`
	StartLine *int `json:"start_line,omitempty"`
	StartCol  *int `json:"start_col,omitempty"`
	EndLine   *int `json:"end_line,omitempty"`
	EndCol    *int `json:"end_col,omitempty"`
	NumStmt   *int `json:"num_stmt,omitempty"`
	Count     *int `json:"count,omitempty"`
}

type BlockCoverages []*BlockCoverage

type Processor interface {
	Name() string
	ParseReport(path string) (*Coverage, string, error)
}

func New() *Coverage {
	return &Coverage{
		Files: FileCoverages{},
	}
}

func NewFileCoverage(file string) *FileCoverage {
	return &FileCoverage{
		File:    file,
		Total:   0,
		Covered: 0,
		Blocks:  BlockCoverages{},
		cache:   map[int]BlockCoverages{},
	}
}

func (coverages FileCoverages) FindByFile(file string) (*FileCoverage, error) {
	for _, c := range coverages {
		if c.File == file {
			return c, nil
		}
	}
	return nil, fmt.Errorf("file name not found: %s", file)
}

func (coverages FileCoverages) FuzzyFindByFile(file string) (*FileCoverage, error) {
	for _, c := range coverages {
		if strings.Contains(strings.TrimLeft(c.File, "./"), strings.TrimLeft(file, "./")) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("file name not found: %s", file)
}

func (fc *FileCoverage) FindBlocksByLine(n int) BlockCoverages {
	if len(fc.cache) == 0 {
		for _, b := range fc.Blocks {
			for i := *b.StartLine; i <= *b.EndLine; i++ {
				fc.cache[i] = append(fc.cache[i], b)
			}
		}
	}
	blocks, ok := fc.cache[n]
	if ok {
		return blocks
	} else {
		return BlockCoverages{}
	}
}
