package coverage

import "fmt"

type Type string

const (
	TypeLOC       Type = "loc"
	TypeStatement Type = "statement"
)

type Coverage struct {
	Type    Type          `json:"type"`
	Format  string        `json:"format"`
	Total   int           `json:"total"`
	Covered int           `json:"covered"`
	Files   FileCoverages `json:"files"`
}

type FileCoverage struct {
	FileName string `json:"file"`
	Total    int    `json:"total"`
	Covered  int    `json:"covered"`
}

type FileCoverages []*FileCoverage

type Processor interface {
	Name() string
	ParseReport(path string) (*Coverage, error)
}

func New() *Coverage {
	return &Coverage{
		Files: FileCoverages{},
	}
}

func NewFileCoverage(fileName string) *FileCoverage {
	return &FileCoverage{
		FileName: fileName,
		Total:    0,
		Covered:  0,
	}
}

func (coverages FileCoverages) FindByFileName(fileName string) (*FileCoverage, error) {
	for _, c := range coverages {
		if c.FileName == fileName {
			return c, nil
		}
	}
	return nil, fmt.Errorf("file name not found: %s", fileName)
}

func Measure(path string) (*Coverage, error) {
	// gocover
	if cov, err := NewGocover().ParseReport(path); err == nil {
		return cov, nil
	}
	// lcov
	if cov, err := NewLcov().ParseReport(path); err == nil {
		return cov, nil
	}
	// simplecov
	if cov, err := NewSimplecov().ParseReport(path); err == nil {
		return cov, nil
	}
	// clover
	if cov, err := NewClover().ParseReport(path); err == nil {
		return cov, nil
	}
	return nil, fmt.Errorf("coverage report not found: %s", path)
}
