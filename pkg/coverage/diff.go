package coverage

type DiffCoverage struct {
	A         float64           `json:"a"`
	B         float64           `json:"b"`
	Diff      float64           `json:"diff"`
	CoverageA *Coverage         `json:"-"`
	CoverageB *Coverage         `json:"-"`
	Files     DiffFileCoverages `json:"files"`
}

type DiffFileCoverage struct {
	File          string        `json:"file"`
	A             float64       `json:"a"`
	B             float64       `json:"b"`
	Diff          float64       `json:"diff"`
	FileCoverageA *FileCoverage `json:"-"`
	FileCoverageB *FileCoverage `json:"-"`
}

type DiffFileCoverages []*DiffFileCoverage

func (c *Coverage) Compare(c2 *Coverage) *DiffCoverage {
	d := &DiffCoverage{
		CoverageA: c,
		CoverageB: c2,
		Files:     DiffFileCoverages{},
	}
	var (
		coverA, coverB float64
	)
	if c != nil && c.Total != 0 {
		coverA = float64(c.Covered) / float64(c.Total) * 100
	}
	if c2 != nil && c2.Total != 0 {
		coverB = float64(c2.Covered) / float64(c2.Total) * 100
	}
	d.A = coverA
	d.B = coverB
	d.Diff = coverB - coverA

	m := map[string]*DiffFileCoverage{}
	if c != nil {
		for _, fc := range c.Files {
			m[fc.File] = &DiffFileCoverage{
				File:          fc.File,
				FileCoverageA: fc,
			}
		}
	}
	if c2 != nil {
		for _, fc := range c2.Files {
			dfc, ok := m[fc.File]
			if ok {
				dfc.FileCoverageB = fc
			} else {
				m[fc.File] = &DiffFileCoverage{
					File:          fc.File,
					FileCoverageB: fc,
				}
			}
		}
	}
	for _, dfc := range m {
		var coverA, coverB float64
		if dfc.FileCoverageA != nil && dfc.FileCoverageA.Total != 0 {
			coverA = float64(dfc.FileCoverageA.Covered) / float64(dfc.FileCoverageA.Total) * 100
		}
		if dfc.FileCoverageB != nil && dfc.FileCoverageB.Total != 0 {
			coverB = float64(dfc.FileCoverageB.Covered) / float64(dfc.FileCoverageB.Total) * 100
		}
		dfc.A = coverA
		dfc.B = coverB
		dfc.Diff = coverB - coverA
		d.Files = append(d.Files, dfc)
	}

	return d
}
