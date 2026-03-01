package coverage

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zhangyunhao116/skipmap"
)

// suffixIndex maps filename → list of absolute paths with that filename.
type suffixIndex map[string][]string

func buildSuffixIndex(fsFiles []string) suffixIndex {
	idx := make(suffixIndex, len(fsFiles))
	for _, f := range fsFiles {
		base := filepath.Base(f)
		idx[base] = append(idx[base], f)
	}
	return idx
}

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
	Type Type   `json:"type"`
	File string `json:"file"`
	// NormalizedPath holds the git-root-relative path resolved via filesystem suffix matching.
	// Different coverage formats produce inconsistent File paths (e.g., Go cover uses module paths
	// like "github.com/user/repo/cmd/main.go", LCOV uses relative paths like "src/utils/groups.ts").
	// NormalizedPath unifies them to git-root-relative paths (e.g., "cmd/main.go") so that
	// matching, merging, comparing, and excluding work correctly across formats.
	// File retains the original parser-produced path for backward compatibility with stored report.json.
	NormalizedPath string         `json:"normalized_path,omitempty"`
	Total          int            `json:"total"`
	Covered        int            `json:"covered"`
	Blocks         BlockCoverages `json:"blocks,omitempty"`
	cache          map[int]BlockCoverages
}

// EffectivePath returns NormalizedPath if set, otherwise File.
func (fc *FileCoverage) EffectivePath() string {
	if fc.NormalizedPath != "" {
		return fc.NormalizedPath
	}
	return fc.File
}

// NormalizePaths populates NormalizedPath for each file in Coverage.
// root is the absolute path to the git root directory.
// fsFiles are absolute paths of files found on the filesystem.
func (c *Coverage) NormalizePaths(root string, fsFiles []string) {
	if c == nil || len(fsFiles) == 0 || root == "" {
		return
	}
	root = filepath.Clean(root)
	idx := buildSuffixIndex(fsFiles)
	for _, fc := range c.Files {
		fc.NormalizedPath = normalizeSingle(root, fc.File, idx)
	}
}

func normalizeSingle(root, file string, idx suffixIndex) string {
	p := filepath.FromSlash(file)

	// Absolute path within root → relative to root
	if filepath.IsAbs(p) {
		rel, err := filepath.Rel(root, p)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
		// Absolute but outside root → try suffix match below
	}

	// Suffix-match against filesystem files by comparing path segments from the end
	base := filepath.Base(p)
	candidates, ok := idx[base]
	if !ok {
		return ""
	}

	cleanFile := filepath.Clean(strings.TrimPrefix(p, "./"))
	fileParts := splitPath(cleanFile)

	var best string
	bestMatchLen := 0
	for _, cand := range candidates {
		rel, err := filepath.Rel(root, cand)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		candParts := splitPath(cand)

		// Count matching segments from the end
		matchLen := 0
		for fi, ci := len(fileParts)-1, len(candParts)-1; fi >= 0 && ci >= 0; fi, ci = fi-1, ci-1 {
			if fileParts[fi] != candParts[ci] {
				break
			}
			matchLen++
		}
		if matchLen == 0 {
			continue
		}

		if matchLen > bestMatchLen || (matchLen == bestMatchLen && len(rel) < len(best)) {
			best = rel
			bestMatchLen = matchLen
		}
	}

	if best != "" {
		return filepath.ToSlash(best)
	}
	return ""
}

func splitPath(p string) []string {
	return strings.Split(filepath.Clean(p), string(filepath.Separator))
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

func NewFileCoverage(file string, coverageType Type) *FileCoverage { //nostyle:repetition
	return &FileCoverage{
		File:    file,
		Type:    coverageType,
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
		if c.EffectivePath() == file || c.File == file {
			return c, nil
		}
	}
	return nil, fmt.Errorf("file name not found: %s", file)
}

func (fc FileCoverages) FuzzyFindByFile(file string) (*FileCoverage, error) { //nostyle:recvtype
	var match *FileCoverage
	for _, c := range fc {
		ep := c.EffectivePath()
		// When coverages are recorded with absolute path. ( ex. /path/to/owner/repo/target.go
		if strings.HasSuffix(strings.TrimLeft(ep, "./"), strings.TrimLeft(file, "./")) {
			if match == nil || len(match.EffectivePath()) > len(ep) {
				match = c
			}
			continue
		}
		// When coverages are recorded in the package path. ( ex. org/repo/package/path/to/Target.kt
		if !filepath.IsAbs(ep) && strings.HasSuffix(file, ep) {
			if match == nil || len(match.EffectivePath()) > len(ep) {
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
		if !filepath.IsAbs(c.File) && strings.HasSuffix(file, c.File) {
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
	m := skipmap.NewInt[*skipmap.IntMap[int]]()

	for _, c := range bc {
		sl := *c.StartLine
		el := *c.EndLine
		for i := sl; i <= el; i++ {
			var mm *skipmap.IntMap[int]
			mm, ok := m.Load(i)
			if !ok {
				mm = skipmap.NewInt[int]()
			}
			m.Store(i, mm)

			if c.Type == TypeLOC || (sl < i && i < el) {
				// TypeLOC or TypeStmt
				mm.Range(func(key int, v int) bool {
					mm.Store(key, v+*c.Count)
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
			mm.Range(func(key int, v int) bool {
				pos = append(pos, key)
				counts = append(counts, v)
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
				mm.Range(func(key int, v int) bool {
					if key >= *c.StartCol {
						mm.Store(key, v+*c.Count)
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
						mm.Store(j, v+*c.Count)
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
				mm.Range(func(key int, v int) bool {
					if key <= *c.EndCol {
						mm.Store(key, v+*c.Count)
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
	m.Range(func(line int, mm *skipmap.IntMap[int]) bool {
		lc := &LineCoverage{
			Line:         line,
			Count:        0,
			PosCoverages: PosCoverages{},
		}
		mm.Range(func(pos int, c int) bool {
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
