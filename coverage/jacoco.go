package coverage

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

var _ Processor = (*Jacoco)(nil)

var JacocoDefaultPath = []string{"build", "reports", "jacoco", "test", "jacocoTestReport.xml"}

type Jacoco struct{}

type JacocoReport struct {
	XMLName     xml.Name `xml:"report"`
	Text        string   `xml:",chardata"`
	Name        string   `xml:"name,attr"`
	Sessioninfo []struct {
		Text  string `xml:",chardata"`
		ID    string `xml:"id,attr"`
		Start string `xml:"start,attr"`
		Dump  string `xml:"dump,attr"`
	} `xml:"sessioninfo"`
	Package []*JacocoReportPackage `xml:"package"`
	Counter []*JacocoReportCounter `xml:"counter"`
}

type JacocoReportPackage struct {
	Text  string `xml:",chardata"`
	Name  string `xml:"name,attr"`
	Class []struct {
		Text           string `xml:",chardata"`
		Name           string `xml:"name,attr"`
		Sourcefilename string `xml:"sourcefilename,attr"`
		Method         []struct {
			Text    string                 `xml:",chardata"`
			Name    string                 `xml:"name,attr"`
			Desc    string                 `xml:"desc,attr"`
			Line    int                    `xml:"line,attr"`
			Counter []*JacocoReportCounter `xml:"counter"`
		} `xml:"method"`
		Counter []*JacocoReportCounter `xml:"counter"`
	} `xml:"class"`
	Sourcefile []struct {
		Text string `xml:",chardata"`
		Name string `xml:"name,attr"`
		Line []struct {
			Text string `xml:",chardata"`
			Nr   int    `xml:"nr,attr"` // line number
			Mi   int    `xml:"mi,attr"` // missed interactions
			Ci   int    `xml:"ci,attr"` // covered interactions
			Mb   int    `xml:"mb,attr"` // missed branches
			Cb   int    `xml:"cb,attr"` // covered branches
		} `xml:"line"`
		Counter []*JacocoReportCounter `xml:"counter"`
	} `xml:"sourcefile"`
	Counter []*JacocoReportCounter `xml:"counter"`
}

type JacocoReportCounter struct {
	Text    string `xml:",chardata"`
	Type    string `xml:"type,attr"`
	Missed  int    `xml:"missed,attr"`
	Covered int    `xml:"covered,attr"`
}

func NewJacoco() *Jacoco {
	return &Jacoco{}
}

func (c *Jacoco) Name() string {
	return "JaCoCo"
}

func (c *Jacoco) ParseReport(path string) (*Coverage, string, error) {
	rp, err := c.detectReportPath(path)
	if err != nil {
		return nil, "", err
	}
	b, err := os.ReadFile(filepath.Clean(rp))
	if err != nil {
		return nil, "", err
	}
	r := JacocoReport{}
	if err := xml.Unmarshal(b, &r); err != nil {
		return nil, "", err
	}
	if len(r.Package) == 0 {
		return nil, "", fmt.Errorf("%s is not Jacoco format", filepath.Clean(rp))
	}

	cov := New()
	cov.Type = TypeLOC
	cov.Format = c.Name()

	flm := map[string]BlockCoverages{}
	// The package name is part of the key, so a key repeats only when the same <package> name
	// appears more than once, as in a report aggregated from several modules. Keep the order of
	// first appearance instead of ranging over flm, whose iteration order Go randomizes, which
	// would churn the files array of a stored report on every run. Merging through
	// Files.FindByFile in one pass, the way lcov.go does, would drop the map, but it rescans
	// the slice once per <sourcefile> element.
	var order []string
	for _, p := range r.Package {
		for _, s := range p.Sourcefile {
			// A class in the default package has <package name="">, and joining that with a
			// separator unconditionally yields "/Foo.java", which filepath.IsAbs reports as
			// absolute on Unix. FuzzyFindByFile then refuses to read it as a package path, so
			// the file never resolves against the pull request's changed files.
			n := s.Name
			if p.Name != "" {
				n = fmt.Sprintf("%s/%s", p.Name, s.Name)
			}
			// A <sourcefile> with no name would key an entry on "", which resolves nothing
			// and only clutters the file list, so none is made.
			if n == "" {
				continue
			}
			f, ok := flm[n]
			if !ok {
				f = BlockCoverages{}
				order = append(order, n)
			}
			for _, l := range s.Line {
				sl := l.Nr
				el := l.Nr
				c := ExecCount(0)
				if l.Ci > 0 {
					c = 1
				}
				f = append(f, &BlockCoverage{
					Type:      TypeLOC,
					StartLine: &sl,
					EndLine:   &el,
					Count:     &c,
				})
			}
			flm[n] = f
		}
	}

	for _, f := range order {
		blocks := flm[f]
		fcov := NewFileCoverage(f, TypeLOC)
		fcov.Blocks = blocks
		// A <sourcefile> lists each line once, but a repeated <package> name brings the same
		// file back with lines that may repeat, so fold the blocks per line the way
		// Coverage.reCalc does rather than counting one line per block.
		lcs := blocks.ToLineCoverages()
		fcov.Total = lcs.Total()
		fcov.Covered = lcs.Covered()
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
		cov.Files = append(cov.Files, fcov)
	}

	return cov, rp, nil
}

func (c *Jacoco) detectReportPath(path string) (string, error) {
	p, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if p.IsDir() {
		np := filepath.Join(path, filepath.Join(JacocoDefaultPath...))
		if _, err := os.Stat(np); err != nil {
			np = filepath.Join(path, JacocoDefaultPath[len(JacocoDefaultPath)-1])
			if _, err := os.Stat(np); err != nil {
				return "", err
			}
		}
		path = np
	}
	return path, nil
}
