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
		lcs := f.Blocks.ToLineCoverages()
		f.Total = lcs.Total()
		f.Covered = lcs.Covered()
		total += f.Total
		covered += f.Covered
	}
	c.Total = total
	c.Covered = covered

	return nil
}
