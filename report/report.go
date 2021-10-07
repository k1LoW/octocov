package report

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal"
	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/pkg/ratio"
	"github.com/olekukonko/tablewriter"
)

const filesHideMin = 30
const filesSkipMax = 100

type Report struct {
	Repository        string             `json:"repository"`
	Ref               string             `json:"ref"`
	Commit            string             `json:"commit"`
	Coverage          *coverage.Coverage `json:"coverage,omitempty"`
	CodeToTestRatio   *ratio.Ratio       `json:"code_to_test_ratio,omitempty"`
	TestExecutionTime *float64           `json:"test_execution_time,omitempty"`
	Timestamp         time.Time          `json:"timestamp"`
	// coverage report path
	rp string
}

func New() (*Report, error) {
	repo := os.Getenv("GITHUB_REPOSITORY")
	ref := os.Getenv("GITHUB_REF")
	if ref == "" {
		b, err := os.ReadFile(".git/HEAD")
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
	}, nil
}

func (r *Report) String() string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (r *Report) Bytes() []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	return b
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
	return strings.Replace(buf.String(), "---|", "--:|", len(h))
}

func (r *Report) Out(w io.Writer) error {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"", makeHeadTitle(r.Ref, r.Commit, r.rp)})
	table.SetAutoFormatHeaders(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})

	if r.Coverage != nil {
		table.Rich([]string{"Coverage", fmt.Sprintf("%.1f%%", r.CoveragePercent())}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	if r.CodeToTestRatio != nil {
		table.Rich([]string{"Code to Test Ratio", fmt.Sprintf("1:%.1f", r.CodeToTestRatioRatio())}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	if r.TestExecutionTime != nil {
		table.Rich([]string{"Test Execution Time", time.Duration(*r.TestExecutionTime).String()}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	table.Render()
	return nil
}

func (r *Report) FileCoveagesTable(files []*gh.PullRequestFile) string {
	if r.Coverage == nil {
		return ""
	}
	if len(files) == 0 {
		return ""
	}
	var t, c int
	exist := false
	rows := [][]string{}
	for _, f := range files {
		fc, err := r.Coverage.Files.FuzzyFindByFile(f.Filename)
		if err != nil {
			continue
		}
		exist = true
		c += fc.Covered
		t += fc.Total
		cover := float64(fc.Covered) / float64(fc.Total) * 100
		if fc.Total == 0 {
			cover = 0.0
		}
		rows = append(rows, []string{fmt.Sprintf("[%s](%s)", f.Filename, f.BlobURL), fmt.Sprintf("%.1f%%", cover)})
	}
	if !exist {
		return ""
	}
	coverAll := float64(c) / float64(t) * 100
	if t == 0 {
		coverAll = 0.0
	}
	title := fmt.Sprintf("### Code coverage of files in pull request scope (%.1f%%)", coverAll)

	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("%s\n\n", title))

	if len(rows) > filesSkipMax {
		buf.WriteString(fmt.Sprintf("Skip file coverages because there are too many files (%d)\n", len(rows)))
		return buf.String()
	}

	if len(rows) > filesHideMin {
		buf.WriteString("<details>\n\n")
	}

	table := tablewriter.NewWriter(buf)
	h := []string{"Files", "Coverage"}
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	for _, v := range rows {
		table.Append(v)
	}
	table.Render()

	if len(rows) > filesHideMin {
		buf.WriteString("\n</details>\n")
	}

	return strings.Replace(strings.Replace(buf.String(), "---|", "--:|", len(h)), "--:|", "---|", 1)
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
	cov, rp, cerr := challengeParseReport(path)
	if cerr != nil {
		f, err := os.Stat(path)
		if err != nil || f.IsDir() {
			return cerr
		}
		b, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, r); err != nil {
			return cerr
		}
		r.rp = path
		return nil
	}
	gitRoot, err := internal.TraverseGitPath(path)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("%s/", filepath.ToSlash(gitRoot))
	for _, f := range cov.Files {
		rel, err := filepath.Rel(prefix, filepath.ToSlash(f.File))
		if err != nil {
			f.File = rel
		}
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
	if r.Coverage == nil || r.Coverage.Total == 0 {
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
		return fmt.Errorf("coverage report '%s' (env %s) is not set", "repository", "GITHUB_REPOSITORY")
	}
	if r.Ref == "" {
		return fmt.Errorf("coverage report '%s' (env %s) is not set", "ref", "GITHUB_REF")
	}
	if r.Commit == "" {
		return fmt.Errorf("coverage report '%s' (env %s) is not set", "commit", "GITHUB_SHA")
	}
	return nil
}

func (r *Report) Compare(r2 *Report) *DiffReport {
	d := &DiffReport{
		RepositoryA: r.Repository,
		RepositoryB: r2.Repository,
		RefA:        r.Ref,
		RefB:        r2.Ref,
		CommitA:     r.Commit,
		CommitB:     r2.Commit,
		ReportA:     r,
		ReportB:     r2,
	}
	if r.Coverage != nil {
		d.Coverage = r.Coverage.Compare(r2.Coverage)
	}
	if r.CodeToTestRatio != nil {
		d.CodeToTestRatio = r.CodeToTestRatio.Compare(r2.CodeToTestRatio)
	}
	if r.TestExecutionTime != nil {
		dt := &DiffTestExecutionTime{
			A:                  r.TestExecutionTime,
			B:                  r2.TestExecutionTime,
			TestExecutionTimeA: r.TestExecutionTime,
			TestExecutionTimeB: r2.TestExecutionTime,
		}
		var t1, t2 float64
		t1 = *r.TestExecutionTime
		if r2.TestExecutionTime != nil {
			t2 = *r2.TestExecutionTime
		}
		dt.Diff = t2 - t1
		d.TestExecutionTime = dt
	}
	return d
}

func makeHeadTitle(ref, commit, rp string) string {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	if strings.HasPrefix(ref, "refs/pull/") {
		ref = strings.Replace(strings.TrimSuffix(ref, "/head"), "refs/pull/", "#", 1)
	}
	if len(commit) > 7 {
		commit = commit[:7]
	} else {
		commit = "-"
	}
	if ref == "" {
		return rp
	}
	return fmt.Sprintf("%s (%s)", ref, commit)
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

func challengeParseReport(path string) (*coverage.Coverage, string, error) {
	// gocover
	if cov, rp, err := coverage.NewGocover().ParseReport(path); err == nil {
		return cov, rp, nil
	}
	// lcov
	if cov, rp, err := coverage.NewLcov().ParseReport(path); err == nil {
		return cov, rp, nil
	}
	// simplecov
	if cov, rp, err := coverage.NewSimplecov().ParseReport(path); err == nil {
		return cov, rp, nil
	}
	// clover
	if cov, rp, err := coverage.NewClover().ParseReport(path); err == nil {
		return cov, rp, nil
	}
	// cobertura
	if cov, rp, err := coverage.NewCobertura().ParseReport(path); err == nil {
		return cov, rp, nil
	}
	return nil, "", fmt.Errorf("coverage report not found: %s", path)
}
