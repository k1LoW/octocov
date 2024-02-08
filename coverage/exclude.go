package coverage

import (
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func (c *Coverage) Exclude(exclude []string) error {
	if len(exclude) == 0 {
		return c.reCalc()
	}

	// Exclude files
	var files FileCoverages
	for i, f := range c.Files {
		excluded := false
		for _, e := range exclude {
			not := false
			if strings.HasPrefix(e, "!") {
				e = strings.TrimPrefix(e, "!")
				not = true
			}
			match, err := doublestar.Match(e, f.File)
			if err != nil {
				return err
			}
			if match {
				if not {
					excluded = false
				} else {
					excluded = true
				}
			}
		}
		if !excluded {
			files = append(files, c.Files[i])
		}
	}
	c.Files = files

	return c.reCalc()
}
