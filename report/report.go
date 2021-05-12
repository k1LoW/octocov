package report

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/pkg/ratio"
)

type Report struct {
	Repository      string             `json:"repository"`
	Ref             string             `json:"ref"`
	Commit          string             `json:"commit"`
	Coverage        *coverage.Coverage `json:"coverage"`
	CodeToTestRatio *ratio.Ratio       `json:"code_to_test_ratio,omitempty"`
	Timestamp       time.Time          `json:"timestamp"`
}

func New() *Report {
	repo := os.Getenv("GITHUB_REPOSITORY")
	ref := os.Getenv("GITHUB_REF")
	if ref == "" {
		b, err := ioutil.ReadFile(".git/HEAD")
		if err == nil {
			splitted := strings.Split(strings.TrimSuffix(string(b), "\n"), " ")
			ref = splitted[1]
		}
	}
	commit := os.Getenv("GITHUB_SHA")
	if commit == "" {
		cmd := exec.Command("git", "rev-parse", "HEAD")
		b, err := cmd.Output()
		if err == nil {
			commit = strings.TrimSuffix(string(b), "\n")
		}
	}

	return &Report{
		Repository: repo,
		Ref:        ref,
		Commit:     commit,
		Timestamp:  time.Now().UTC(),
	}
}

func (r *Report) String() string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (r *Report) MeasureCoverage(path string) error {
	cov, cerr := coverage.Measure(path)
	if cerr != nil {
		b, err := ioutil.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, r); err != nil {
			return cerr
		}
		return nil
	}
	r.Coverage = cov
	return nil
}

func (r *Report) MeasureCodeToTestRatio(code, test []string) error {
	ratio, err := ratio.Measure(".", code, test)
	if err != nil {
		return err
	}
	r.CodeToTestRatio = ratio
	return nil
}

func (r *Report) Validate() error {
	if r.Repository == "" {
		return fmt.Errorf("coverage report '%s' is not set", "repository")
	}
	if r.Ref == "" {
		return fmt.Errorf("coverage report '%s' is not set", "ref")
	}
	if r.Commit == "" {
		return fmt.Errorf("coverage report '%s' is not set", "commit")
	}
	return nil
}

func (r *Report) CoveragePercent() float64 {
	return float64(r.Coverage.Covered) / float64(r.Coverage.Total) * 100
}

func (r *Report) CodeToTestRatioRatio() float64 {
	return float64(r.CodeToTestRatio.Test) / float64(r.CodeToTestRatio.Code)
}
