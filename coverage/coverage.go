package coverage

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zhangyunhao116/skipmap"
)

type Type string

const (
	TypeLOC    Type = "loc"
	TypeStmt   Type = "statement"
	TypeMerged Type = "merged"
)

type Coverage struct {
	Type    Type          `json:"type"`
	Format  string        `json:"format"`
	Total   int           `json:"total"`
	Covered int           `json:"covered"`
	Files   FileCoverages `json:"files"`
}

type FileCoverage struct {
	File    string         `json:"file"`
	Total   int            `json:"total"`
	Covered int            `json:"covered"`
	Blocks  BlockCoverages `json:"blocks,omitempty"`
	cache   map[int]BlockCoverages
}

type FileCoverages []*FileCoverage

type BlockCoverage struct {
	Type      Type `json:"type"`
	StartLine *int `json:"start_line,omitempty"`
	StartCol  *int `json:"start_col,omitempty"`
	EndLine   *int `json:"end_line,omitempty"`
	EndCol    *int `json:"end_col,omitempty"`
	NumStmt   *int `json:"num_stmt,omitempty"`
	Count     *int `json:"count,omitempty"`
}

type BlockCoverages []*BlockCoverage

type Processor interface {
	Name() string
	ParseReport(path string) (*Coverage, string, error)
}

func New() *Coverage {
	return &Coverage{
		Files: FileCoverages{},
	}
}

func NewFileCoverage(file string) *FileCoverage { //nostyle:repetition
	return &FileCoverage{
		File:    file,
		Total:   0,
		Covered: 0,
		Blocks:  BlockCoverages{},
		cache:   map[int]BlockCoverages{},
	}
}

func (c *Coverage) DeleteBlockCoverages() {
	for _, f := range c.Files {
		f.Blocks = BlockCoverages{}
	}
}

func (fc FileCoverages) FindByFile(file string) (*FileCoverage, error) { //nostyle:recvtype
	for _, c := range fc {
		if c.File == file {
			return c, nil
		}
	}
	return nil, fmt.Errorf("file name not found: %s", file)
}

func (fc FileCoverages) FuzzyFindByFile(file string) (*FileCoverage, error) { //nostyle:recvtype
	var match *FileCoverage
	for _, c := range fc {
		// When coverages are recorded with absolute path. ( ex. /path/to/owner/repo/target.go
		if strings.HasSuffix(strings.TrimLeft(c.File, "./"), strings.TrimLeft(file, "./")) {
			if match == nil || len(match.File) > len(c.File) {
				match = c
			}
			continue
		}
		// When coverages are recorded in the package path. ( ex. org/repo/package/path/to/Target.kt
		if !strings.HasPrefix(c.File, "/") && strings.HasSuffix(file, c.File) {
			if match == nil || len(match.File) > len(c.File) {
				match = c
			}
			continue
		}
	}
	if match != nil {
		return match, nil
	}
	return nil, fmt.Errorf("file name not found: %s", file)
}

func (fc FileCoverages) PathPrefix() (string, error) { //nostyle:recvtype
	if len(fc) == 0 {
		return "", errors.New("no file coverages")
	}
	p := strings.Split(filepath.Dir(filepath.ToSlash(fc[0].File)), "/")
	for _, c := range fc {
		d := strings.Split(filepath.Dir(filepath.ToSlash(c.File)), "/")
		i := 0
		for {
			if len(p) <= i {
				break
			}
			if len(d) <= i {
				break
			}
			if p[i] != d[i] {
				break
			}
			i += 1
		}
		p = p[:i]
	}
	s := strings.Join(p, "/")
	if s == "" && strings.HasPrefix(fc[0].File, "/") {
		s = "/"
	}
	if s == "." {
		s = ""
	}
	return s, nil
}

func (fc *FileCoverage) FindBlocksByLine(n int) BlockCoverages {
	if fc == nil {
		return BlockCoverages{}
	}
	if len(fc.cache) == 0 {
		fc.cache = map[int]BlockCoverages{}
		for _, b := range fc.Blocks {
			for i := *b.StartLine; i <= *b.EndLine; i++ {
				_, ok := fc.cache[i]
				if !ok {
					fc.cache[i] = BlockCoverages{}
				}
				fc.cache[i] = append(fc.cache[i], b)
			}
		}
	}
	blocks, ok := fc.cache[n]
	if ok {
		return blocks
	} else {
		return BlockCoverages{}
	}
}

func (dc DiffFileCoverages) FuzzyFindByFile(file string) (*DiffFileCoverage, error) { //nostyle:recvtype
	var match *DiffFileCoverage
	for _, c := range dc {
		// When coverages are recorded with absolute path. ( ex. /path/to/owner/repo/target.go
		if strings.HasSuffix(strings.TrimLeft(c.File, "./"), strings.TrimLeft(file, "./")) {
			if match == nil || len(match.File) > len(c.File) {
				match = c
			}
			continue
		}
		// When coverages are recorded in the package path. ( ex. org/repo/package/path/to/Target.kt
		if !strings.HasPrefix(c.File, "/") && strings.HasSuffix(file, c.File) {
			if match == nil || len(match.File) > len(c.File) {
				match = c
			}
			continue
		}
	}
	if match != nil {
		return match, nil
	}
	return nil, fmt.Errorf("file name not found: %s", file)
}

func (bc BlockCoverages) MaxCount() int { //nostyle:recvtype
	counts := map[int]int{}
	for _, c := range bc {
		sl := *c.StartLine
		el := *c.EndLine
		for i := sl; i <= el; i++ {
			_, ok := counts[i]
			if !ok {
				counts[i] = 0
			}
			counts[i] += *c.Count
		}
	}
	max := 0
	for _, v := range counts {
		if v > max {
			max = v
		}
	}
	return max
}

const (
	startPos = -1
	endPos   = 99999
)

type PosCoverage struct {
	Pos   int
	Count int
}

type PosCoverages []*PosCoverage

func (pc PosCoverages) FindCountByPos(pos int) (int, error) { //nostyle:recvtype
	before := PosCoverages{}
	after := PosCoverages{}
	for _, c := range pc {
		switch {
		case c.Pos < pos:
			before = append(before, c)
		case c.Pos == pos:
			return c.Count, nil
		case c.Pos > pos:
			after = append(after, c)
		}
	}

	if len(before) == 0 || len(after) == 0 {
		return 0, fmt.Errorf("count not found: %d", pos)
	}

	if before[len(before)-1].Pos != startPos && after[0].Pos != endPos {
		return 0, fmt.Errorf("count not found: %d", pos)
	}

	if before[len(before)-1].Pos == startPos {
		return before[len(before)-1].Count, nil
	}

	if after[0].Pos == endPos {
		return after[0].Count, nil
	}

	return 0, errors.New("invalid pos")
}

type LineCoverage struct {
	Line         int
	Count        int
	PosCoverages PosCoverages
}

type LineCoverages []*LineCoverage

func (lc LineCoverages) FindByLine(l int) (*LineCoverage, error) { //nostyle:recvtype
	for _, c := range lc {
		if c.Line == l {
			return c, nil
		}
	}
	return nil, fmt.Errorf("no line coverage: %d", l)
}

func (lc LineCoverages) Total() int { //nostyle:recvtype
	return len(lc)
}

func (lc LineCoverages) Covered() int { //nostyle:recvtype
	covered := 0
	for _, c := range lc {
		if c.Count > 0 {
			covered += 1
		}
	}
	return covered
}

func (bc BlockCoverages) ToLineCoverages() LineCoverages { //nostyle:recvtype
	m := skipmap.NewInt()

	for _, c := range bc {
		sl := *c.StartLine
		el := *c.EndLine
		for i := sl; i <= el; i++ {
			var mm *skipmap.IntMap
			v, ok := m.Load(i)
			if ok {
				mm, ok = v.(*skipmap.IntMap)
				if !ok {
					panic("invalid type") //nostyle:dontpanic
				}
			} else {
				mm = skipmap.NewInt()
			}
			m.Store(i, mm)

			if c.Type == TypeLOC || (sl < i && i < el) {
				// TypeLOC or TypeStmt
				mm.Range(func(key int, v any) bool {
					mm.Store(key, v.(int)+*c.Count)
					return true
				})
				if _, ok := mm.Load(startPos); !ok {
					mm.Store(startPos, *c.Count)
				}
				if _, ok := mm.Load(endPos); !ok {
					mm.Store(endPos, *c.Count)
				}
				continue
			}

			// TypeStmt
			startCount := 0
			endCount := 0
			startTo := startPos
			endFrom := endPos
			var (
				pos    []int
				counts []int
			)
			mm.Range(func(key int, v any) bool {
				pos = append(pos, key)
				counts = append(counts, v.(int))
				return true
			})

			if len(pos) > 1 && pos[0] == startPos {
				startCount = counts[0]
				startTo = pos[1] - 1
			}

			if len(pos) > 1 && pos[len(pos)-1] == endPos {
				endCount = counts[len(pos)-1]
				endFrom = pos[len(pos)-2] + 1
			}

			switch {
			case i == sl && i != el:
				mm.Range(func(key int, v any) bool {
					if key >= *c.StartCol {
						mm.Store(key, v.(int)+*c.Count)
					}
					return true
				})
				if _, ok := mm.Load(*c.StartCol); !ok {
					mm.Store(*c.StartCol, *c.Count)
				}
				if _, ok := mm.Load(endPos); !ok {
					mm.Store(endPos, *c.Count)
				}
			case i == sl && i == el:
				for j := *c.StartCol; j <= *c.EndCol; j++ {
					v, ok := mm.Load(j)
					if ok {
						mm.Store(j, v.(int)+*c.Count)
					} else {
						if j <= startTo {
							mm.Store(j, startCount+*c.Count)
						} else if endFrom <= j {
							mm.Store(j, endCount+*c.Count)
						} else {
							mm.Store(j, *c.Count)
						}
					}
				}
			case i != sl && i == el:
				mm.Range(func(key int, v any) bool {
					if key <= *c.EndCol {
						mm.Store(key, v.(int)+*c.Count)
					}
					return true
				})
				if _, ok := mm.Load(startPos); !ok {
					mm.Store(startPos, *c.Count)
				}
				if _, ok := mm.Load(*c.EndCol); !ok {
					mm.Store(*c.EndCol, *c.Count)
				}
			}
		}
	}

	lcs := LineCoverages{}
	m.Range(func(line int, mmi any) bool {
		mm, ok := mmi.(*skipmap.IntMap)
		if !ok {
			return false
		}
		lc := &LineCoverage{
			Line:         line,
			Count:        0,
			PosCoverages: PosCoverages{},
		}
		mm.Range(func(pos int, ci any) bool {
			c, ok := ci.(int)
			if !ok {
				return false
			}
			lc.PosCoverages = append(lc.PosCoverages, &PosCoverage{
				Pos:   pos,
				Count: c,
			})
			if c > lc.Count {
				lc.Count = c
			}
			return true
		})
		lcs = append(lcs, lc)
		return true
	})

	return lcs
}
