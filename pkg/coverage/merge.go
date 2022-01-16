package coverage

import (
	"errors"
)

func (c *Coverage) Merge(c2 *Coverage) error {
	{
		deleted := true
		for _, f := range c.Files {
			if len(f.Blocks) > 0 {
				deleted = false
			}
		}
		if len(c.Files) > 0 && deleted {
			return errors.New("can not merge: BlockCoverages are already deleted.")
		}
	}
	{
		deleted := true
		for _, f := range c2.Files {
			if len(f.Blocks) > 0 {
				deleted = false
			}
			fc, err := c.Files.FindByFile(f.File)
			if err == nil {
				fc.Blocks = append(fc.Blocks, f.Blocks...)
			} else {
				c.Files = append(c.Files, f)
			}
		}
		if len(c2.Files) > 0 && deleted {
			return errors.New("can not merge: BlockCoverages are already deleted.")
		}
	}
	if c.Type != c2.Type {
		c.Type = TypeMerged
	}

	total := 0
	covered := 0
	for _, f := range c.Files {
		lcs := f.Blocks.ToLineCoverages()
		for _, lc := range lcs {
			total += 1
			if lc.Count > 0 {
				covered += 1
			}
		}
	}
	c.Total = total
	c.Covered = covered

	return nil
}
