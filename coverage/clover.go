package coverage

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Path    string   `xml:"path,attr"`
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
	// A file can be described at the project level and again inside a <package>, so elements
	// naming the same file stack onto one entry rather than being listed and counted twice, the
	// shape #724 described for LCOV. The lookup goes through a map rather than Files.FindByFile
	// for the reason cobertura.go and jacoco.go record, that FindByFile rescans the slice once
	// per element, and a Clover report has one <file> element per file, so a large report
	// parsed in quadratic time. The map holds only identities that are unique by construction,
	// a path attribute or an absolute name. A bare relative name is a display name two files can
	// share, and stacking on it fused different files, so such an element is appended as before.
	byIdentity := map[string]*FileCoverage{}
	add := func(f CloverReportFile) {
		identity, unique := cloverIdentity(f)
		if unique {
			if existing, ok := byIdentity[identity]; ok {
				existing.Blocks = append(existing.Blocks, parseReportLines(f)...)
				return
			}
		}
		fcov := NewFileCoverage(identity, TypeLOC)
		fcov.Blocks = parseReportLines(f)
		cov.Files = append(cov.Files, fcov)
		if unique {
			byIdentity[identity] = fcov
		}
	}
	for _, f := range r.Project.File {
		add(f)
	}
	for _, p := range r.Project.Package {
		for _, f := range p.File {
			add(f)
		}
	}
	for _, fcov := range cov.Files {
		// Fold per line through foldLines rather than trusting the <metrics> attributes, so the
		// total ParseReport returns agrees with the blocks returned beside it, the contract #728
		// and #734 settled for the other LOC parsers and #738 records. The report's own statement
		// count is the more authoritative number in principle, but no consumer ever read it. Every
		// path recounted from the blocks when #749 made this choice, and since #738 reCalc re-sums
		// this fold instead, so the total folded here is the one every consumer sees.
		fcov.foldLines()
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
	}
	return cov, rp, nil
}

// cloverIdentity returns the path an element is filed under and whether that path is unique by
// construction. The path attribute is the real location and name the display name, typically a
// basename (#639), so path wins when it is given and name is relative. An absolute name is a
// real location too. A bare relative name is not, since files in different directories share it.
func cloverIdentity(f CloverReportFile) (string, bool) {
	if f.Path != "" && !isAbsReportPath(f.Name) {
		return f.Path, true
	}
	return f.Name, isAbsReportPath(f.Name)
}

// isAbsReportPath reports whether a path a report recorded is absolute on the host that produced
// the report. filepath.IsAbs answers for the host octocov runs on, so a Unix path read on Windows,
// or a Windows path read on Unix, came out relative there, and a file was filed under the wrong
// identity or refused a merge depending on the runner rather than on the report. It is not
// consulted at all, since its Windows answer has also widened between Go releases.
func isAbsReportPath(p string) bool {
	if strings.HasPrefix(p, "/") {
		return true
	}
	// A drive letter and a separator, the one shape of a Windows absolute path a report writes.
	return len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/') &&
		(('a' <= p[0] && p[0] <= 'z') || ('A' <= p[0] && p[0] <= 'Z'))
}

func parseReportLines(f CloverReportFile) BlockCoverages {
	blocks := BlockCoverages{}
	for _, l := range f.Line {
		if l.Type != "stmt" {
			continue
		}
		sl := l.Num
		el := l.Num
		c := toExecCount(l.Count)
		blocks = append(blocks, &BlockCoverage{
			Type:      TypeLOC,
			StartLine: &sl,
			EndLine:   &el,
			Count:     &c,
		})
	}
	return blocks
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
