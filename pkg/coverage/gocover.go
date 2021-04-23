package coverage

import (
	"os"
	"path/filepath"

	"golang.org/x/tools/cover"
)

const GocoverDefaultPath = "coverage.out"

type Gocover struct{}

func NewGocover() *Gocover {
	return &Gocover{}
}

func (g *Gocover) ParseReport(path string) (*Coverage, error) {
	rp, err := g.detectReportPath(path)
	if err != nil {
		return nil, err
	}
	profiles, err := cover.ParseProfiles(rp)
	if err != nil {
		return nil, err
	}
	cov := New()
	cov.Type = TypeStatement
	cov.Format = "Go coverage"
	for _, p := range profiles {
		total, covered := g.countProfile(p)
		fcov := NewFileCoverage(p.FileName)
		fcov.Total = total
		fcov.Covered = covered
		cov.Total += total
		cov.Covered += covered
		cov.Files = append(cov.Files, fcov)
	}
	return cov, nil
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
