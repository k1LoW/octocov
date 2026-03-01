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
	d.Diff = coverA - coverB

	// m maps path keys to DiffFileCoverage. A single DiffFileCoverage may be
	// registered under multiple keys (EffectivePath and File) so that lookups
	// succeed even when only one side has NormalizedPath set.
	m := map[string]*DiffFileCoverage{}
	var dfcList []*DiffFileCoverage
	if c != nil {
		for _, fc := range c.Files {
			ep := fc.EffectivePath()
			dfc := &DiffFileCoverage{
				File:          ep,
				FileCoverageA: fc,
			}
			m[ep] = dfc
			if fc.File != ep {
				m[fc.File] = dfc
			}
			dfcList = append(dfcList, dfc)
		}
	}
	if c2 != nil {
		for _, fc := range c2.Files {
			ep := fc.EffectivePath()
			dfc := lookupDiffMap(m, ep, fc.File)
			if dfc != nil {
				dfc.FileCoverageB = fc
			} else {
				dfc = &DiffFileCoverage{
					File:          ep,
					FileCoverageB: fc,
				}
				m[ep] = dfc
				if fc.File != ep {
					m[fc.File] = dfc
				}
				dfcList = append(dfcList, dfc)
			}
		}
	}
	for _, dfc := range dfcList {
		var coverA, coverB float64
		if dfc.FileCoverageA != nil && dfc.FileCoverageA.Total != 0 {
			coverA = float64(dfc.FileCoverageA.Covered) / float64(dfc.FileCoverageA.Total) * 100
		}
		if dfc.FileCoverageB != nil && dfc.FileCoverageB.Total != 0 {
			coverB = float64(dfc.FileCoverageB.Covered) / float64(dfc.FileCoverageB.Total) * 100
		}
		dfc.A = coverA
		dfc.B = coverB
		dfc.Diff = coverA - coverB
		d.Files = append(d.Files, dfc)
	}

	return d
}

// lookupDiffMap tries to find an existing DiffFileCoverage by effectivePath first, then by file.
func lookupDiffMap(m map[string]*DiffFileCoverage, effectivePath, file string) *DiffFileCoverage {
	if dfc, ok := m[effectivePath]; ok {
		return dfc
	}
	if file != effectivePath {
		if dfc, ok := m[file]; ok {
			return dfc
		}
	}
	return nil
}
