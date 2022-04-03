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
	if r.Package == nil {
		return nil, "", fmt.Errorf("%s is not Jacoco format", filepath.Clean(rp))
	}

	cov := New()
	cov.Type = TypeLOC
	cov.Format = c.Name()

	flm := map[string]BlockCoverages{}
	for _, p := range r.Package {
		for _, s := range p.Sourcefile {
			n := fmt.Sprintf("%s/%s", p.Name, s.Name)
			f, ok := flm[n]
			if !ok {
				f = BlockCoverages{}
			}
			for _, l := range s.Line {
				sl := l.Nr
				el := l.Nr
				c := 0
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
