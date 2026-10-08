package coverage

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var _ Processor = (*Cobertura)(nil)

const CoberturaDefaultPath = "coverage.xml"

// conditionCoverageRe reads the branches a line took out of its condition-coverage attribute,
// ex. "50% (1/2)".
var conditionCoverageRe = regexp.MustCompile(`\((\d+)/(\d+)\)`)

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
					Number            int    `xml:"number,attr"`
					Hits              int    `xml:"hits,attr"`
					Branch            string `xml:"branch,attr"`
					ConditionCoverage string `xml:"condition-coverage,attr"`
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
	fbm := map[string]BranchCoverages{}
	// A file can be split over several <class> elements (e.g. one class per inner class), so
	// keep the order of first appearance instead of ranging over flm, whose iteration order Go
	// randomizes and which would churn the files array of a stored report on every run.
	// Merging through Files.FindByFile in one pass, the way lcov.go does, would drop the map,
	// but it rescans the slice once per <class> element and this format has one per class.
	var order []string
	for _, p := range r.Packages.Package {
		for _, c := range p.Classes.Class {
			n := c.Filename
			f, ok := flm[n]
			if !ok {
				f = BlockCoverages{}
				order = append(order, n)
			}
			for _, l := range c.Lines.Line {
				sl := l.Number
				el := l.Number
				c := toExecCount(l.Hits)
				f = append(f, &BlockCoverage{
					Type:      TypeLOC,
					StartLine: &sl,
					EndLine:   &el,
					Count:     &c,
				})
				if b, ok := parseCoberturaBranch(l.Number, l.Branch, l.ConditionCoverage); ok {
					fbm[n] = append(fbm[n], b)
				}
			}
			flm[n] = f
		}
	}

	for _, f := range order {
		blocks := flm[f]
		fcov := NewFileCoverage(f, TypeLOC)
		fcov.Blocks = blocks
		// Two <class> elements of one file can both list the same line (a Kotlin or Scala
		// lambda shows up under its outer class and under a synthetic one), so fold the
		// blocks per line through foldLines, whose result reCalc re-sums instead of recomputing.
		// Counting one line per block would count such a line twice and return a total the
		// blocks do not support.
		fcov.foldLines()
		// The same line under two <class> elements reports the same branches twice, which
		// foldBranches counts once. The branch-rate attributes of the report are not read,
		// for the same reason the line totals are folded from the lines.
		fcov.Branches = fbm[f]
		fcov.foldBranches()
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
		cov.BranchTotal += fcov.BranchTotal
		cov.BranchCovered += fcov.BranchCovered
		cov.Files = append(cov.Files, fcov)
	}

	return cov, rp, nil
}

// parseCoberturaBranch reads the branches of a line with branch="true". A line whose
// condition-coverage is missing or does not read as "(covered/total)" is skipped rather than
// failing the report, since the line itself is still counted.
func parseCoberturaBranch(number int, branch, conditionCoverage string) (*BranchCoverage, bool) {
	if branch != "true" {
		return nil, false
	}
	m := conditionCoverageRe.FindStringSubmatch(conditionCoverage)
	if len(m) != 3 {
		return nil, false
	}
	covered, err := strconv.Atoi(m[1])
	if err != nil {
		return nil, false
	}
	total, err := strconv.Atoi(m[2])
	if err != nil || total == 0 {
		return nil, false
	}
	return &BranchCoverage{
		Line:    number,
		Total:   total,
		Covered: min(covered, total),
	}, true
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
