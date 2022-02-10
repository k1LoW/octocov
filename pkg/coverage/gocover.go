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
	cov.Type = TypeStmt
	cov.Format = g.Name()
	for _, p := range profiles {
		fcov := NewFileCoverage(p.FileName)
		for _, b := range p.Blocks {
			sl := b.StartLine
			sc := b.StartCol
			el := b.EndLine
			ec := b.EndCol
			ns := b.NumStmt
			c := b.Count
			fcov.Blocks = append(fcov.Blocks, &BlockCoverage{
				Type:      TypeStmt,
				StartLine: &sl,
				StartCol:  &sc,
				EndLine:   &el,
				EndCol:    &ec,
				NumStmt:   &ns,
				Count:     &c,
			})
		}
		lc := fcov.Blocks.ToLineCoverages()
		fcov.Total = lc.Total()
		fcov.Covered = lc.Covered()
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
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
