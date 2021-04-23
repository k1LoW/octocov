package coverage

import "fmt"

type Type string

const (
	TypeLOC       Type = "loc"
	TypeStatement      = "statement"
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

func (coverages FileCoverages) FindByFileName(fileName string) (*FileCoverage, error) {
	for _, c := range coverages {
		if c.FileName == fileName {
			return c, nil
		}
	}
	return nil, fmt.Errorf("file name not found: %s", fileName)
}

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
