package coverage

import (
	"os"
	"path/filepath"

	"golang.org/x/tools/cover"
)

var _ Processor = (*Gocover)(nil)

const GocoverDefaultPath = "coverage.out"

type Gocover struct{}

func NewGocover() *Gocover {
	return &Gocover{}
}

func (g *Gocover) Name() string {
	return "Go coverage"
}

func (g *Gocover) ParseReport(path string) (*Coverage, string, error) {
	rp, err := g.detectReportPath(path)
	if err != nil {
		return nil, "", err
	}
	profiles, err := cover.ParseProfiles(rp)
	if err != nil {
		return nil, "", err
	}
	cov := New()
	cov.Type = TypeStatement
	cov.Format = g.Name()
	for _, p := range profiles {
		total, covered := g.countProfile(p)
		fcov := NewFileCoverage(p.FileName)
		fcov.Total = total
		fcov.Covered = covered
		for _, b := range p.Blocks {
			sl := b.StartLine
			sc := b.StartCol
			el := b.EndLine
			ec := b.EndCol
			ns := b.NumStmt
			c := b.Count
			fcov.Blocks = append(fcov.Blocks, &BlockCoverage{
				Type:      TypeStatement,
				StartLine: &sl,
				StartCol:  &sc,
				EndLine:   &el,
				EndCol:    &ec,
				NumStmt:   &ns,
				Count:     &c,
			})
		}
		cov.Total += total
		cov.Covered += covered
		cov.Files = append(cov.Files, fcov)
	}
	return cov, rp, nil
}

func (g *Gocover) detectReportPath(path string) (string, error) {
	p, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if p.IsDir() {
		path = filepath.Join(path, GocoverDefaultPath)
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func (g *Gocover) countProfile(p *cover.Profile) (int, int) {
	var total, covered int
	for _, b := range p.Blocks {
		total += b.NumStmt
		if b.Count > 0 {
			covered += b.NumStmt
		}
	}
	if total == 0 {
		return 0, 0
	}
	return total, covered
}
