package coverage

func (c *Coverage) Merge(c2 *Coverage) error {
	if c2 == nil {
		return c.reCalc()
	}
	// Type
	switch {
	case c2.Type == "":
	case c.Type != TypeLOC || c2.Type != TypeLOC:
		// If either is not LOC, merge as Merged
		c.Type = TypeMerged
	}
	// Files
	for _, fc2 := range c2.Files {
		fc, err := c.Files.FindByFile(fc2.File)
		if err == nil {
			if fc2.Type != fc.Type {
				fc.Type = TypeMerged
			}
			// Merged coverage should be counted as LOC as duplicate blocks may be stacked.
			fc.Blocks = append(fc.Blocks, fc2.Blocks...)
		} else {
			c.Files = append(c.Files, fc2)
		}
	}
	return c.reCalc()
}

func (c *Coverage) reCalc() error {
	total := 0
	covered := 0
	for _, f := range c.Files {
		var fileTotal, fileCovered int

		switch c.Type {
		case TypeLOC, TypeMerged:
			lcs := f.Blocks.ToLineCoverages()
			fileTotal = lcs.Total()
			fileCovered = lcs.Covered()

		case TypeStmt: // Coverage of a single unmerged TypeStmt.
			for _, b := range f.Blocks {
				fileTotal += *b.NumStmt
				if *b.Count > 0 {
					fileCovered += *b.NumStmt
				}
			}
		}

		f.Total = fileTotal
		f.Covered = fileCovered
		total += fileTotal
		covered += fileCovered
	}
	c.Total = total
	c.Covered = covered

	return nil
}
