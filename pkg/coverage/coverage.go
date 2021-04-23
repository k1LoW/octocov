package coverage

import "fmt"

type Type int

const (
	TypeLOC Type = iota + 1
	TypeStatement
)

var typeNames = [...]string{"", "loc", "statement"}

func (t Type) String() string {
	return typeNames[t]
}

type Coverage struct {
	Type    Type
	Format  string
	Total   int
	Covered int
	Files   FileCoverages
}

type FileCoverage struct {
	FileName string
	Total    int
	Covered  int
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
