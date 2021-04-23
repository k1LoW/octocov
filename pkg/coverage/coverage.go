package coverage

type Type int

const (
	TypeLine Type = iota + 1
	TypeStatement
)

var typeNames = [...]string{"", "line", "statement"}

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

type Processor interface {
	Name() string
	ParseReport(path string) (*Coverage, error)
}

func New() *Coverage {
	return &Coverage{
		Type:    TypeStatement,
		Format:  "Golang txt",
		Total:   0,
		Covered: 0,
		Files:   FileCoverages{},
	}
}

func NewFileCoverage(fileName string) *FileCoverage {
	return &FileCoverage{
		FileName: fileName,
		Total:    0,
		Covered:  0,
	}
}
