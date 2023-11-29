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
	var (
		fileName       string
		total, covered int
	)
	cov := New()
	cov.Type = TypeLOC
	cov.Format = l.Name()
	parsed := false
	blocks := BlockCoverages{}
	for scanner.Scan() {
		l := scanner.Text()
		if l == "end_of_record" {
			fcov, err := cov.Files.FindByFile(fileName)
			if err != nil {
				fcov = NewFileCoverage(fileName)
			}
			fcov.Total += total
			fcov.Covered += covered
			fcov.Blocks = blocks
			cov.Total += total
			cov.Covered += covered
			cov.Files = append(cov.Files, fcov)
			total = 0
			covered = 0
			parsed = true
			blocks = BlockCoverages{}
			continue
		}
		splitted := strings.Split(l, ":")
		if len(splitted) != 2 {
			continue
		}
		switch splitted[0] {
		case "SF":
			fileName = splitted[1]
		case "DA":
			total += 1
			nums := strings.Split(splitted[1], ",")
			if len(nums) != 2 {
				_ = r.Close() //nostyle:handlerrors
				return nil, "", fmt.Errorf("can not parse: %s", l)
			}
			line, err := strconv.Atoi(nums[0])
			if err != nil {
				_ = r.Close() //nostyle:handlerrors
				return nil, "", err
			}
			count, err := strconv.Atoi(nums[1])
			if err != nil {
				_ = r.Close() //nostyle:handlerrors
				return nil, "", err
			}
			if count > 0 {
				covered += 1
			}
			blocks = append(blocks, &BlockCoverage{
				Type:      TypeLOC,
				StartLine: &line,
				EndLine:   &line,
				Count:     &count,
			})
		default:
			// not implemented
		}
	}
	if err := r.Close(); err != nil {
		return nil, "", err
	}
	if !parsed {
		return nil, "", errors.New("can not parse")
	}
	return cov, rp, nil
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
