package coverage

import "sort"

// PatchFileCoverage represents the coverage of the changed lines of a single file.
type PatchFileCoverage struct {
	File    string `json:"file"`
	Total   int    `json:"total"`
	Covered int    `json:"covered"`
}

// Rate returns the coverage rate (0-100) of the changed lines. Returns 0 if there are no changed lines.
func (p *PatchFileCoverage) Rate() float64 {
	if p.Total == 0 {
		return 0
	}
	return float64(p.Covered) / float64(p.Total) * 100
}

// PatchCoverage represents the coverage of the changed lines across multiple files (e.g. a pull request).
type PatchCoverage struct {
	Total   int                  `json:"total"`
	Covered int                  `json:"covered"`
	Files   []*PatchFileCoverage `json:"files,omitempty"`
}

// Rate returns the coverage rate (0-100) of the changed lines. Returns 0 if there are no changed lines.
func (p *PatchCoverage) Rate() float64 {
	if p.Total == 0 {
		return 0
	}
	return float64(p.Covered) / float64(p.Total) * 100
}

// PatchCoverage calculates the coverage of the changed lines of a single file.
func (fc *FileCoverage) PatchCoverage(changedLines []int) *PatchFileCoverage {
	covered := 0
	for _, line := range changedLines {
		for _, b := range fc.FindBlocksByLine(line) {
			if b.Count != nil && *b.Count > 0 {
				covered++
				break
			}
		}
	}
	return &PatchFileCoverage{
		File:    fc.EffectivePath(),
		Total:   len(changedLines),
		Covered: covered,
	}
}

// PatchCoverage calculates the coverage of the changed lines across the given files.
// changedFiles maps a file path to the line numbers changed in that file.
// Files that cannot be matched against this Coverage are skipped.
func (c *Coverage) PatchCoverage(changedFiles map[string][]int) *PatchCoverage {
	pc := &PatchCoverage{}
	for file, changedLines := range changedFiles {
		if len(changedLines) == 0 {
			continue
		}
		fc, err := c.Files.FuzzyFindByFile(file)
		if err != nil || fc == nil {
			continue
		}
		fpc := fc.PatchCoverage(changedLines)
		if fpc.Total == 0 {
			continue
		}
		pc.Total += fpc.Total
		pc.Covered += fpc.Covered
		pc.Files = append(pc.Files, fpc)
	}
	sort.Slice(pc.Files, func(i, j int) bool { return pc.Files[i].File < pc.Files[j].File })
	return pc
}
