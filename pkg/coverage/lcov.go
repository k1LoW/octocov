package coverage

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var LcovDefaultPath = []string{"coverage", "lcov.info"}

type Lcov struct{}

func NewLcov() *Lcov {
	return &Lcov{}
}

func (l *Lcov) ParseReport(path string) (*Coverage, error) {
	rp, err := l.detectReportPath(path)
	if err != nil {
		return nil, err
	}
	r, err := os.Open(filepath.Clean(rp))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Close()
	}()
	scanner := bufio.NewScanner(r)
	var (
		fileName       string
		total, covered int
	)
	cov := New()
	cov.Type = TypeLOC
	cov.Format = "LCOV"
	parsed := false
	for scanner.Scan() {
		l := scanner.Text()
		if l == "end_of_record" {
			fcov, err := cov.Files.FindByFileName(fileName)
			if err != nil {
				fcov = NewFileCoverage(fileName)
			}
			fcov.Total += total
			fcov.Covered += covered
			cov.Total += total
			cov.Covered += covered
			cov.Files = append(cov.Files, fcov)
			total = 0
			covered = 0
			parsed = true
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
			if !strings.HasSuffix(splitted[1], ",0") {
				covered += 1
			}
		default:
			// not implemented
		}
	}
	if !parsed {
		return nil, errors.New("can not parse")
	}
	return cov, nil
}

func (s *Lcov) detectReportPath(path string) (string, error) {
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
