package report

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/pkg/ratio"
	"github.com/olekukonko/tablewriter"
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

func (r *Report) Table() string {
	h := []string{}
	m := []string{}
	if r.Coverage != nil {
		h = append(h, "Coverage")
		m = append(m, fmt.Sprintf("%.1f%%", r.CoveragePercent()))
	}
	if r.CodeToTestRatio != nil {
		h = append(h, "Code to Test Ratio")
		m = append(m, fmt.Sprintf("1:%.1f", r.CodeToTestRatioRatio()))
	}
	if r.TestExecutionTime != nil {
		h = append(h, "Test Execution Time")
		d := time.Duration(*r.TestExecutionTime)
		m = append(m, d.String())
	}
	buf := new(bytes.Buffer)
	table := tablewriter.NewWriter(buf)
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Append(m)
	table.Render()
	return buf.String()
}

func (r *Report) CountMeasured() int {
	c := 0
	if r.IsMeasuredCoverage() {
		c += 1
	}
	if r.IsMeasuredCodeToTestRatio() {
		c += 1
	}
	if r.IsMeasuredTestExecutionTime() {
		c += 1
	}
	return c
}

func (r *Report) IsMeasuredCoverage() bool {
	return r.Coverage != nil
}

func (r *Report) IsMeasuredCodeToTestRatio() bool {
	return r.CodeToTestRatio != nil
}

func (r *Report) IsMeasuredTestExecutionTime() bool {
	return r.TestExecutionTime != nil
}

func (r *Report) MeasureCoverage(path string) error {
	cov, rp, cerr := coverage.Measure(path)
	if cerr != nil {
		f, err := os.Stat(path)
		if err != nil || f.IsDir() {
			return cerr
		}
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

func (r *Report) MeasureTestExecutionTime(ctx context.Context, stepNames []string) error {
	if r.Repository == "" {
		return fmt.Errorf("env %s is not set", "GITHUB_REPOSITORY")
	}
	splitted := strings.Split(r.Repository, "/")
	owner := splitted[0]
	repo := splitted[1]
	g, err := gh.New()
	if err != nil {
		return err
	}
	if len(stepNames) > 0 {
		steps := []gh.Step{}
		for _, n := range stepNames {
			s, err := g.GetStepsByName(ctx, owner, repo, n)
			if err != nil {
				return err
			}
			steps = append(steps, s...)
		}
		d := mergeExecutionTimes(steps)
		t := float64(d)
		r.TestExecutionTime = &t
		return nil
	}
	fi, err := os.Stat(r.rp)
	if err != nil {
		return err
	}
	jobID, err := g.DetectCurrentJobID(ctx, owner, repo)
	if err != nil {
		return err
	}
	d, err := g.GetStepExecutionTimeByTime(ctx, owner, repo, jobID, fi.ModTime())
	if err != nil {
		return err
	}
	t := float64(d)
	r.TestExecutionTime = &t
	return nil
}

func (r *Report) CoveragePercent() float64 {
	if r.Coverage.Total == 0 {
		return 0.0
	}
	return float64(r.Coverage.Covered) / float64(r.Coverage.Total) * 100
}

func (r *Report) CodeToTestRatioRatio() float64 {
	if r.CodeToTestRatio.Code == 0 {
		return 0.0
	}
	return float64(r.CodeToTestRatio.Test) / float64(r.CodeToTestRatio.Code)
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

type timePoint struct {
	t time.Time
	c int
}

func mergeExecutionTimes(steps []gh.Step) time.Duration {
	timePoints := []timePoint{}
	for _, s := range steps {
		timePoints = append(timePoints, timePoint{s.StartedAt, 1}, timePoint{s.CompletedAt, -1})
	}
	sort.Slice(timePoints, func(i, j int) bool { return timePoints[i].t.UnixNano() < timePoints[j].t.UnixNano() })
	var st, ct time.Time
	d := time.Duration(0)
	c := 0
	for _, tp := range timePoints {
		if c == 0 {
			st = tp.t
		}
		c += tp.c
		if c == 0 {
			ct = tp.t
			d += ct.Sub(st)
		}
	}
	return d
}
