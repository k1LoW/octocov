package coverage

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

var _ Processor = (*Clover)(nil)

const CloverDefaultPath = "coverage.xml"

type Clover struct{}

type CloverReport struct {
	XMLName   xml.Name             `xml:"coverage"`
	Generated string               `xml:"generated,attr"`
	Project   *CloverReportProject `xml:"project"`
}

type CloverReportProject struct {
	Timestamp string                `xml:"timestamp,attr"`
	File      []CloverReportFile    `xml:"file"`
	Package   []CloverReportPackage `xml:"package"`
	Metrics   struct {
		Files               int `xml:"files,attr"`
		Loc                 int `xml:"loc,attr"`
		Ncloc               int `xml:"ncloc,attr"`
		Classes             int `xml:"classes,attr"`
		Methods             int `xml:"methods,attr"`
		Coveredmethods      int `xml:"coveredmethods,attr"`
		Conditionals        int `xml:"conditionals,attr"`
		Coveredconditionals int `xml:"coveredconditionals,attr"`
		Statements          int `xml:"statements,attr"`
		Coveredstatements   int `xml:"coveredstatements,attr"`
		Elements            int `xml:"elements,attr"`
		Coveredelements     int `xml:"coveredelements,attr"`
	} `xml:"metrics"`
}

type CloverReportPackage struct {
	XMLName xml.Name           `xml:"package"`
	Name    string             `xml:"name,attr"`
	File    []CloverReportFile `xml:"file"`
}

type CloverReportFile struct {
	XMLName xml.Name `xml:"file"`
	Name    string   `xml:"name,attr"`
	Metrics struct {
		Loc                 int `xml:"loc,attr"`
		Ncloc               int `xml:"ncloc,attr"`
		Classes             int `xml:"classes,attr"`
		Methods             int `xml:"methods,attr"`
		Coveredmethods      int `xml:"coveredmethods,attr"`
		Conditionals        int `xml:"conditionals,attr"`
		Coveredconditionals int `xml:"coveredconditionals,attr"`
		Statements          int `xml:"statements,attr"`
		Coveredstatements   int `xml:"coveredstatements,attr"`
		Elements            int `xml:"elements,attr"`
		Coveredelements     int `xml:"coveredelements,attr"`
	} `xml:"metrics"`
	Class struct {
		Name      string `xml:"name,attr"`
		Namespace string `xml:"namespace,attr"`
		Metrics   struct {
			Complexity          int `xml:"complexity,attr"`
			Methods             int `xml:"methods,attr"`
			Coveredmethods      int `xml:"coveredmethods,attr"`
			Conditionals        int `xml:"conditionals,attr"`
			Coveredconditionals int `xml:"coveredconditionals,attr"`
			Statements          int `xml:"statements,attr"`
			Coveredstatements   int `xml:"coveredstatements,attr"`
			Elements            int `xml:"elements,attr"`
			Coveredelements     int `xml:"coveredelements,attr"`
		} `xml:"metrics"`
	} `xml:"class"`
	Line []struct {
		Num        int     `xml:"num,attr"`
		Type       string  `xml:"type,attr"`
		Name       string  `xml:"name,attr"`
		Visibility string  `xml:"visibility,attr"`
		Complexity int     `xml:"complexity,attr"`
		Crap       float64 `xml:"crap,attr"`
		Count      int     `xml:"count,attr"`
	} `xml:"line"`
}

func NewClover() *Clover {
	return &Clover{}
}

func (c *Clover) Name() string {
	return "Clover"
}

func (c *Clover) ParseReport(path string) (*Coverage, string, error) {
	rp, err := c.detectReportPath(path)
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(filepath.Clean(rp))
	if err != nil {
		return nil, "", err
	}
	r := CloverReport{}
	if err := xml.Unmarshal(b, &r); err != nil {
		return nil, "", err
	}
	if r.Project == nil {
		return nil, "", fmt.Errorf("%s is not Clover format", filepath.Clean(rp))
	}

	cov := New()
	// ref: https://openclover.org/doc/manual/latest/general--about-code-coverage.html
	// > As Clover uses source code instrumentation, it actually "sees" a real code structure.
	// > Therefore, Clover offers a Statement Coverage metric, which is similar to a Line Coverage metric in terms of it's granularity and precision.
	cov.Type = TypeLOC
	cov.Format = c.Name()
	for _, f := range r.Project.File {
		fcov := parseReportFile(f)
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
		cov.Files = append(cov.Files, fcov)
	}
	for _, p := range r.Project.Package {
		for _, f := range p.File {
			fcov := parseReportFile(f)
			cov.Total += fcov.Total
			cov.Covered += fcov.Covered
			cov.Files = append(cov.Files, fcov)
		}
	}
	return cov, rp, nil
}

func parseReportFile(f CloverReportFile) *FileCoverage {
	fcov := NewFileCoverage(f.Name)
	fcov.Covered = f.Metrics.Coveredstatements
	fcov.Total = f.Metrics.Statements
	for _, l := range f.Line {
		if l.Type != "stmt" {
			continue
		}
		sl := l.Num
		el := l.Num
		c := l.Count
		fcov.Blocks = append(fcov.Blocks, &BlockCoverage{
			Type:      TypeLOC,
			StartLine: &sl,
			EndLine:   &el,
			Count:     &c,
		})
	}
	return fcov
}

func (c *Clover) detectReportPath(path string) (string, error) {
	p, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if p.IsDir() {
		path = filepath.Join(path, CloverDefaultPath)
	}
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}
