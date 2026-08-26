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
// Changed lines that belong to no coverage block are not instrumented by the coverage format
// (blank lines, comments, package/import declarations, closing braces, ...), and are excluded
// from the total, as with the instrumented-line counts behind the current/prev metrics.
func (fc *FileCoverage) PatchCoverage(changedLines []int) *PatchFileCoverage {
	if fc == nil {
		return &PatchFileCoverage{}
	}
	total, covered := 0, 0
	for _, line := range changedLines {
		blocks := fc.FindBlocksByLine(line)
		if len(blocks) == 0 {
			continue
		}
		total++
		for _, b := range blocks {
			if b.Count != nil && *b.Count > 0 {
				covered++
				break
			}
		}
	}
	return &PatchFileCoverage{
		File:    fc.EffectivePath(),
		Total:   total,
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
