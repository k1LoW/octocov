package report

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/pkg/ratio"
)

type Report struct {
	Repository        string             `json:"repository"`
	Ref               string             `json:"ref"`
	Commit            string             `json:"commit"`
	Coverage          *coverage.Coverage `json:"coverage"`
	CodeToTestRatio   *ratio.Ratio       `json:"code_to_test_ratio,omitempty"`
	TestExecutionTime *float64           `json:"test_execution_time,omitempty"`

	Timestamp time.Time `json:"timestamp"`
	// coverage report path
	rp string
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
	cov, rp, cerr := coverage.Measure(path)
	if cerr != nil {
		b, err := ioutil.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, r); err != nil {
			return cerr
		}
		r.rp = path
		return nil
	}
	r.Coverage = cov
	r.rp = rp
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

func (r *Report) MeasureTestExecutionTime() error {
	if os.Getenv("GITHUB_RUN_ID") == "" {
		return nil
	}
	fi, err := os.Stat(r.rp)
	if err != nil {
		return err
	}
	splitted := strings.Split(r.Repository, "/")
	owner := splitted[0]
	repo := splitted[1]
	gh, err := gh.New()
	if err != nil {
		return err
	}
	ctx := context.Background()
	jobID, err := gh.DetectCurrentJobID(ctx, owner, repo, nil)
	if err != nil {
		return err
	}
	d, err := gh.GetStepExecutionTimeByTime(ctx, owner, repo, jobID, fi.ModTime())
	if err != nil {
		return err
	}
	t := float64(d)
	r.TestExecutionTime = &t
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
