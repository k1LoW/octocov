package coverage

func (c *Coverage) Merge(c2 *Coverage) error {
	if c2 == nil {
		c2 = &Coverage{}
	}
	// Type
	switch {
	case c2.Type == "":
	case c.Type != TypeLOC || c2.Type != TypeLOC:
		c.Type = TypeMerged
	}
	// Files
	for _, fc2 := range c2.Files {
		fc, err := c.Files.FindByFile(fc2.File)
		if err == nil {
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

		switch f.Type {
		case TypeLOC:
			lcs := f.Blocks.ToLineCoverages()
			fileTotal = lcs.Total()
			fileCovered = lcs.Covered()

		case TypeStmt:
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
