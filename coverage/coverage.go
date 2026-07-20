package coverage

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
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

	FormatMerged = "Merged"
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

// ExecCount is a block execution count. It is uint64 internally so that
// u64-wrapped counters emitted by llvm-based tools (e.g. cargo-llvm-cov when
// profile counters race) survive parsing without truncation.
//
// In stored report.json the canonical field is "count_u64", which always
// carries the raw uint64 value; uint64-aware readers prefer it and fall back
// to "count" for reports written before its introduction. "count" is kept as
// a compatibility field, clamped to MaxInt64 via ExecCount.MarshalJSON,
// because binaries that decode counts into int (upstream octocov, older
// versions of this fork) fail the whole report on out-of-int64 number
// literals; they ignore the unknown "count_u64". The final migration step is
// to stop emitting "count".
type ExecCount uint64

func (c ExecCount) MarshalJSON() ([]byte, error) { //nostyle:recvtype
	if c > math.MaxInt64 {
		c = math.MaxInt64
	}
	return []byte(strconv.FormatUint(uint64(c), 10)), nil
}

// satAdd returns a+b, saturating at MaxUint64 instead of wrapping.
func satAdd(a, b ExecCount) ExecCount {
	if s := a + b; s >= a {
		return s
	}
	return math.MaxUint64
}

// toExecCount converts a count parsed from a signed representation, treating
// negative values as 0.
func toExecCount(c int) ExecCount {
	if c < 0 {
		return 0
	}
	return ExecCount(c)
}

type BlockCoverage struct {
	Type      Type       `json:"type"`
	StartLine *int       `json:"start_line,omitempty"`
	StartCol  *int       `json:"start_col,omitempty"`
	EndLine   *int       `json:"end_line,omitempty"`
	EndCol    *int       `json:"end_col,omitempty"`
	NumStmt   *int       `json:"num_stmt,omitempty"`
	Count     *ExecCount `json:"count,omitempty"`
}

// blockCoverageJSON mirrors BlockCoverage with the canonical "count_u64"
// field alongside the clamped compatibility "count" (see ExecCount).
type blockCoverageJSON struct {
	Type      Type       `json:"type"`
	StartLine *int       `json:"start_line,omitempty"`
	StartCol  *int       `json:"start_col,omitempty"`
	EndLine   *int       `json:"end_line,omitempty"`
	EndCol    *int       `json:"end_col,omitempty"`
	NumStmt   *int       `json:"num_stmt,omitempty"`
	Count     *ExecCount `json:"count,omitempty"`
	CountU64  *uint64    `json:"count_u64,omitempty"`
}

func (bc *BlockCoverage) MarshalJSON() ([]byte, error) {
	a := blockCoverageJSON{
		Type:      bc.Type,
		StartLine: bc.StartLine,
		StartCol:  bc.StartCol,
		EndLine:   bc.EndLine,
		EndCol:    bc.EndCol,
		NumStmt:   bc.NumStmt,
		Count:     bc.Count,
	}
	if bc.Count != nil {
		raw := uint64(*bc.Count)
		a.CountU64 = &raw
	}
	return json.Marshal(a)
}

func (bc *BlockCoverage) UnmarshalJSON(data []byte) error {
	var a blockCoverageJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	bc.Type = a.Type
	bc.StartLine = a.StartLine
	bc.StartCol = a.StartCol
	bc.EndLine = a.EndLine
	bc.EndCol = a.EndCol
	bc.NumStmt = a.NumStmt
	bc.Count = a.Count
	if a.CountU64 != nil {
		c := ExecCount(*a.CountU64)
		bc.Count = &c
	}
	return nil
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

func (bc BlockCoverages) MaxCount() ExecCount { //nostyle:recvtype
	counts := map[int]ExecCount{}
	for _, c := range bc {
		sl := *c.StartLine
		el := *c.EndLine
		for i := sl; i <= el; i++ {
			_, ok := counts[i]
			if !ok {
				counts[i] = 0
			}
			counts[i] = satAdd(counts[i], *c.Count)
		}
	}
	max := ExecCount(0)
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
	Count ExecCount
}

type PosCoverages []*PosCoverage

func (pc PosCoverages) FindCountByPos(pos int) (ExecCount, error) { //nostyle:recvtype
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
	Count        ExecCount
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
	m := skipmap.NewInt[*skipmap.IntMap[ExecCount]]()

	for _, c := range bc {
		sl := *c.StartLine
		el := *c.EndLine
		for i := sl; i <= el; i++ {
			var mm *skipmap.IntMap[ExecCount]
			mm, ok := m.Load(i)
			if !ok {
				mm = skipmap.NewInt[ExecCount]()
			}
			m.Store(i, mm)

			if c.Type == TypeLOC || (sl < i && i < el) {
				// TypeLOC or TypeStmt
				mm.Range(func(key int, v ExecCount) bool {
					mm.Store(key, satAdd(v, *c.Count))
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
			startCount := ExecCount(0)
			endCount := ExecCount(0)
			startTo := startPos
			endFrom := endPos
			var (
				pos    []int
				counts []ExecCount
			)
			mm.Range(func(key int, v ExecCount) bool {
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
				mm.Range(func(key int, v ExecCount) bool {
					if key >= *c.StartCol {
						mm.Store(key, satAdd(v, *c.Count))
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
						mm.Store(j, satAdd(v, *c.Count))
					} else {
						if j <= startTo {
							mm.Store(j, satAdd(startCount, *c.Count))
						} else if endFrom <= j {
							mm.Store(j, satAdd(endCount, *c.Count))
						} else {
							mm.Store(j, *c.Count)
						}
					}
				}
			case i != sl && i == el:
				mm.Range(func(key int, v ExecCount) bool {
					if key <= *c.EndCol {
						mm.Store(key, satAdd(v, *c.Count))
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
	m.Range(func(line int, mm *skipmap.IntMap[ExecCount]) bool {
		lc := &LineCoverage{
			Line:         line,
			Count:        0,
			PosCoverages: PosCoverages{},
		}
		mm.Range(func(pos int, c ExecCount) bool {
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
