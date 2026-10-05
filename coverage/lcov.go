package coverage

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var _ Processor = (*Lcov)(nil)

var LcovDefaultPath = []string{"coverage", "lcov.info"}

type Lcov struct{}

func NewLcov() *Lcov {
	return &Lcov{}
}

func (l *Lcov) Name() string {
	return "LCOV"
}

func (l *Lcov) ParseReport(path string) (*Coverage, string, error) {
	rp, err := l.detectReportPath(path)
	if err != nil {
		return nil, "", err
	}
	r, err := os.Open(filepath.Clean(rp))
	if err != nil {
		return nil, "", err
	}
	scanner := bufio.NewScanner(r)
	var fileName string
	cov := New()
	cov.Type = TypeLOC
	cov.Format = l.Name()
	parsed := false
	blocks := BlockCoverages{}
	branches := newLcovBranches()
	for scanner.Scan() {
		l := scanner.Text()
		if l == "end_of_record" {
			fcov, err := cov.Files.FindByFile(fileName)
			if err != nil {
				fcov = NewFileCoverage(fileName, TypeLOC)
				cov.Files = append(cov.Files, fcov)
			}
			// The same source file can appear in several records (e.g. .info files of
			// separate test runs concatenated together), and a single record can list the
			// same line more than once. Stack the blocks on the file already found, the way
			// Coverage.Merge does, instead of replacing the earlier record and listing the
			// file twice.
			fcov.Blocks = append(fcov.Blocks, blocks...)
			// A record lists each line's branches as a whole, so another record of the file
			// observes the same branches again, and foldBranches takes the larger of them per
			// line rather than adding them up.
			fcov.Branches = append(fcov.Branches, branches.coverages()...)
			parsed = true
			blocks = BlockCoverages{}
			branches = newLcovBranches()
			continue
		}
		// The value may itself contain ':' (e.g. SF:C:\path\to\file), so split only once.
		splitted := strings.SplitN(l, ":", 2)
		if len(splitted) != 2 {
			continue
		}
		switch splitted[0] {
		case "SF":
			fileName = splitted[1]
		case "DA":
			// DA:<line>,<count>[,<checksum>]
			nums := strings.Split(splitted[1], ",")
			if len(nums) != 2 && len(nums) != 3 {
				_ = r.Close() //nostyle:handlerrors
				return nil, "", fmt.Errorf("can not parse: %s", l)
			}
			line, err := strconv.Atoi(nums[0])
			if err != nil {
				_ = r.Close() //nostyle:handlerrors
				return nil, "", err
			}
			// Parse as uint64: llvm-cov can emit u64-wrapped (negative)
			// execution counts when profile counters race (e.g. a thread
			// still running at process exit), so counts up to MaxUint64 are
			// valid input. On ErrRange (> MaxUint64) ParseUint has already
			// saturated the value; accept it instead of rejecting the whole
			// report over one corrupt counter.
			count, err := strconv.ParseUint(nums[1], 10, 64)
			if err != nil && !errors.Is(err, strconv.ErrRange) {
				_ = r.Close() //nostyle:handlerrors
				return nil, "", err
			}
			c := ExecCount(count)
			blocks = append(blocks, &BlockCoverage{
				Type:      TypeLOC,
				StartLine: &line,
				EndLine:   &line,
				Count:     &c,
			})
		case "BRDA":
			// BRDA:<line>,<block>,<branch>,<taken>
			// A line that cannot be read is skipped rather than failing the report, since
			// branches are read on top of the lines.
			branches.add(splitted[1])
		default:
			// not implemented (BRF and BRH are summaries, as LF and LH are)
		}
	}
	if err := r.Close(); err != nil {
		return nil, "", err
	}
	if !parsed {
		return nil, "", errors.New("can not parse")
	}
	// Fold per line through foldLines, whose result reCalc re-sums instead of recomputing,
	// rather than counting one line per block, which would count a line twice when a record
	// repeats it. Folding here once rather
	// than on every end_of_record keeps a file named by R records from being re-folded R
	// times.
	for _, fcov := range cov.Files {
		fcov.foldLines()
		fcov.foldBranches()
		cov.Total += fcov.Total
		cov.Covered += fcov.Covered
		cov.BranchTotal += fcov.BranchTotal
		cov.BranchCovered += fcov.BranchCovered
	}
	return cov, rp, nil
}

// lcovBranches collects the BRDA lines of one record. Within a record each BRDA line is a
// distinct branch of its line, so they add up into one BranchCoverage per line, and a branch
// listed twice under the same block and branch counts once.
type lcovBranches struct {
	lines map[int]*BranchCoverage
	order []int
	taken map[string]bool
}

func newLcovBranches() *lcovBranches {
	return &lcovBranches{
		lines: map[int]*BranchCoverage{},
		taken: map[string]bool{},
	}
}

func (b *lcovBranches) add(v string) {
	// The line comes first and the taken count last. The block may carry a prefix (lcov 2.x
	// writes "e" for an exception branch) and the branch may be an expression, so what lies
	// between is used only to tell branches apart.
	ls, rest, ok := strings.Cut(v, ",")
	if !ok {
		return
	}
	i := strings.LastIndex(rest, ",")
	if i < 0 {
		return
	}
	id, ts := rest[:i], rest[i+1:]
	line, err := strconv.Atoi(ls)
	if err != nil || id == "" {
		return
	}
	// "-" means the block holding the branch was never executed.
	taken := false
	if ts != "-" {
		t, err := strconv.ParseUint(ts, 10, 64)
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			return
		}
		taken = t > 0
	}
	bc, ok := b.lines[line]
	if !ok {
		bc = &BranchCoverage{Line: line}
		b.lines[line] = bc
		b.order = append(b.order, line)
	}
	key := strconv.Itoa(line) + "," + id
	prev, seen := b.taken[key]
	if !seen {
		bc.Total++
	}
	if taken && !prev {
		bc.Covered++
	}
	b.taken[key] = prev || taken
}

func (b *lcovBranches) coverages() BranchCoverages {
	bcs := make(BranchCoverages, 0, len(b.order))
	for _, l := range b.order {
		bcs = append(bcs, b.lines[l])
	}
	return bcs
}

func (l *Lcov) detectReportPath(path string) (string, error) {
	p, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if p.IsDir() {
		// path/to/coverage/lcov.info
		np := filepath.Join(path, LcovDefaultPath[0], LcovDefaultPath[1])
		if _, err := os.Stat(np); err != nil {
			// path/to/lcov.info
			np = filepath.Join(path, LcovDefaultPath[1])
			if _, err := os.Stat(np); err != nil {
				return "", err
			}
		}
		path = np
	}
	return path, nil
}
