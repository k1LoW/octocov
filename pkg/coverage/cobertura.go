package coverage

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

var _ Processor = (*Cobertura)(nil)

const CoberturaDefaultPath = "coverage.xml"

type Cobertura struct{}

type CoberturaReport struct {
	XMLName         xml.Name `xml:"coverage"`
	Version         string   `xml:"version,attr"`
	Timestamp       string   `xml:"timestamp,attr"`
	LinesValid      int      `xml:"lines-valid,attr"`
	LinesCovered    int      `xml:"lines-covered,attr"`
	LineRate        float64  `xml:"line-rate,attr"`
	BranchesCovered int      `xml:"branches-covered,attr"`
	BranchesValid   int      `xml:"branches-valid,attr"`
	BranchRate      float64  `xml:"branch-rate,attr"`
	Complexity      int      `xml:"complexity,attr"`
	Sources         struct {
		Source []string `xml:"source"`
	} `xml:"sources"`
	Packages *CoberturaReportPackages `xml:"packages"`
}

type CoberturaReportPackages struct {
	Package []CoberturaReportPackage `xml:"package"`
}

type CoberturaReportPackage struct {
	Name       string  `xml:"name,attr"`
	LineRate   float64 `xml:"line-rate,attr"`
	BranchRate float64 `xml:"branch-rate,attr"`
	Complexity int     `xml:"complexity,attr"`
	Classes    struct {
		Class []struct {
			Filename   string  `xml:"filename,attr"`
			Complexity int     `xml:"complexity,attr"`
			LineRate   float64 `xml:"line-rate,attr"`
			BranchRate float64 `xml:"branch-rate,attr"`
			Methods    struct {
				Method []struct {
					Name       string  `xml:"name,attr"`
					Signature  string  `xml:"signature,attr"`
					LineRate   float64 `xml:"line-rate,attr"`
					BranchRate float64 `xml:"branch-rate,attr"`
					Lines      struct {
						Line []struct {
							Number int `xml:"number,attr"`
							Hits   int `xml:"hits,attr"`
						} `xml:"line"`
					} `xml:"lines"`
				}
			} `xml:"methods"`
			Lines struct {
				Line []struct {
					Number int `xml:"number,attr"`
					Hits   int `xml:"hits,attr"`
				} `xml:"line"`
			} `xml:"lines"`
		} `xml:"class"`
	} `xml:"classes"`
}

func NewCobertura() *Cobertura {
	return &Cobertura{}
}

func (c *Cobertura) Name() string {
	return "Cobertura"
}

func (c *Cobertura) ParseReport(path string) (*Coverage, string, error) {
	rp, err := c.detectReportPath(path)
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(filepath.Clean(rp))
	if err != nil {
		return nil, "", err
	}
	r := CoberturaReport{}
	if err := xml.Unmarshal(b, &r); err != nil {
		return nil, "", err
	}
	if r.Packages == nil {
		return nil, "", fmt.Errorf("%s is not Cobertura format", filepath.Clean(rp))
	}

	cov := New()
	cov.Type = TypeLOC
	cov.Format = c.Name()

	flm := map[string]BlockCoverages{}
	for _, p := range r.Packages.Package {
		for _, c := range p.Classes.Class {
			n := c.Filename
			f, ok := flm[n]
			if !ok {
				f = BlockCoverages{}
			}
			for _, l := range c.Lines.Line {
				sl := l.Number
				el := l.Number
				ns := 1
				c := l.Hits
				f = append(f, &BlockCoverage{
					Type:      TypeLOC,
					StartLine: &sl,
					EndLine:   &el,
					NumStmt:   &ns,
					Count:     &c,
				})
			}
			flm[n] = f
		}
	}

	for f, blocks := range flm {
		fcov := NewFileCoverage(f)
		for _, b := range blocks {
			fcov.Total += 1
			if *b.Count > 0 {
				fcov.Covered += 1
			}
		}
		fcov.Blocks = blocks
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
		cov.Files = append(cov.Files, fcov)
	}

	return cov, rp, nil
}

func (c *Cobertura) detectReportPath(path string) (string, error) {
	p, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if p.IsDir() {
		path = filepath.Join(path, CoberturaDefaultPath)
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}
