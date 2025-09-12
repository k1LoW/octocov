package report

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/expr-lang/expr"
	"github.com/goccy/go-json"
	"github.com/k1LoW/errors"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/ratio"
	"github.com/olekukonko/tablewriter"
	"github.com/samber/lo"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const filesHideMin = 30
const filesSkipMax = 100

var (
	_ config.Reporter = (*Report)(nil)
)

type Report struct {
	Repository        string             `json:"repository"`
	Ref               string             `json:"ref"`
	Commit            string             `json:"commit"`
	Coverage          *coverage.Coverage `json:"coverage,omitempty"`
	CodeToTestRatio   *ratio.Ratio       `json:"code_to_test_ratio,omitempty"`
	TestExecutionTime *float64           `json:"test_execution_time,omitempty"`
	Timestamp         time.Time          `json:"timestamp"`
	CustomMetrics     []*CustomMetricSet `json:"custom_metrics,omitempty"`

	// coverage report paths
	covPaths []string
	opts     *Options
}

func New(ownerrepo string, opts ...Option) (*Report, error) {
	if ownerrepo == "" {
		ownerrepo = os.Getenv("GITHUB_REPOSITORY")
	}
	ref := os.Getenv("GITHUB_REF")
	if ref == "" {
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		b, err := cmd.Output()
		if err == nil {
			ref = strings.TrimSuffix(string(b), "\n")
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
	o := &Options{}
	for _, setter := range opts {
		setter(o)
	}

	return &Report{
		Repository: ownerrepo,
		Ref:        ref,
		Commit:     commit,
		Timestamp:  time.Now().UTC(),
		opts:       o,
	}, nil
}

func (r *Report) Title() string {
	key := r.Key()
	if key == "" {
		return "Code Metrics Report"
	}
	return fmt.Sprintf("Code Metrics Report (%s)", key)
}

func (r *Report) Key() string {
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		return ""
	}
	if r.Repository == repo {
		return ""
	}
	return strings.TrimPrefix(r.Repository, fmt.Sprintf("%s/", repo))
}

func (r *Report) String() string {
	return string(r.Bytes())
}

func (r *Report) Bytes() []byte {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err) //nostyle:dontpanic
	}
	return b
}

func (r *Report) Table() string {
	var (
		h []string
		m []string
	)
	if r.IsMeasuredCoverage() {
		h = append(h, "Coverage")
		m = append(m, fmt.Sprintf("%.1f%%", floor1(r.CoveragePercent())))
	}
	if r.IsMeasuredCodeToTestRatio() {
		h = append(h, "Code to Test Ratio")
		m = append(m, fmt.Sprintf("1:%.1f", floor1(r.CodeToTestRatioRatio())))
	}
	if r.IsMeasuredTestExecutionTime() {
		h = append(h, "Test Execution Time")
		d := time.Duration(r.TestExecutionTimeNano())
		m = append(m, d.String())
	}
	buf := new(bytes.Buffer)
	table := tablewriter.NewWriter(buf)
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Append(m)
	table.Render()
	return strings.Replace(buf.String(), "---|", "--:|", len(h))
}

func (r *Report) Out(w io.Writer) error {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"", makeHeadTitle(r.Ref, r.Commit, r.covPaths)})
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})

	if r.IsMeasuredCoverage() {
		table.Rich([]string{"Coverage", fmt.Sprintf("%.1f%%", floor1(r.CoveragePercent()))}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	if r.IsMeasuredCodeToTestRatio() {
		table.Rich([]string{"Code to Test Ratio", fmt.Sprintf("1:%.1f", floor1(r.CodeToTestRatioRatio()))}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	if r.IsMeasuredTestExecutionTime() {
		table.Rich([]string{"Test Execution Time", time.Duration(*r.TestExecutionTime).String()}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	table.Render()

	if r.IsCollectedCustomMetrics() {
		for _, m := range r.CustomMetrics {
			if _, err := w.Write([]byte("\n")); err != nil {
				return err
			}
			if err := m.Out(w); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Report) FileCoveragesTable(files []*gh.PullRequestFile) string {
	if r.Coverage == nil {
		return ""
	}
	if len(files) == 0 {
		return ""
	}
	var t, c int
	exist := false
	var rows [][]string
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
		rows = append(rows, []string{fmt.Sprintf("[%s](%s)", f.Filename, f.BlobURL), fmt.Sprintf("%.1f%%", floor1(cover))})
	}
	if !exist {
		return ""
	}
	coverAll := float64(c) / float64(t) * 100
	if t == 0 {
		coverAll = 0.0
	}
	title := fmt.Sprintf("### Code coverage of files in pull request scope (%.1f%%)", floor1(coverAll))

	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "%s\n\n", title)

	if len(rows) > filesSkipMax {
		fmt.Fprintf(buf, "Skip file coverages because there are too many files (%d)\n", len(rows))
		return buf.String()
	}

	if len(rows) > filesHideMin {
		buf.WriteString("<details>\n\n")
	}

	table := tablewriter.NewWriter(buf)
	h := []string{"Files", "Coverage"}
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
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
	c += len(r.CustomMetrics)
	return c
}

func (r *Report) IsMeasuredCoverage() bool {
	return r.Coverage != nil
}

func (r *Report) IsMeasuredCodeToTestRatio() bool {
	return r.CodeToTestRatio != nil
}

func (r *Report) IsMeasuredTestExecutionTime() bool {
	if r == nil {
		return false
	}
	return r.TestExecutionTime != nil
}

func (r *Report) IsCollectedCustomMetrics() bool {
	return len(r.CustomMetrics) > 0
}

func (r *Report) Load(path string) error {
	f, err := os.Stat(path)
	if err != nil || f.IsDir() {
		return fmt.Errorf("octocov report.json not found: %s", path)
	}
	b, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, r); err != nil {
		return err
	}
	r.covPaths = append(r.covPaths, path)
	return nil
}

func (r *Report) MeasureCoverage(patterns, exclude []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("coverage report not found: %s", patterns)
	}
	var paths []string
	for _, pattern := range patterns {
		p, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return err
		}
		paths = append(paths, p...)
	}
	var errs error
	for _, path := range paths {
		cov, rp, err := challengeParseReport(path)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if r.Coverage == nil {
			r.Coverage = cov
		} else {
			if err := r.Coverage.Merge(cov); err != nil {
				return errors.Join(errs, err)
			}
		}
		r.covPaths = append(r.covPaths, rp)
	}

	// fallback load report.json
	if r.Coverage == nil && len(paths) == 1 {
		path := paths[0]
		if err := r.Load(path); err != nil {
			return errors.Join(errs, err)
		}
	}

	if r.Coverage == nil {
		if errs != nil {
			return errs
		}
		return nil
	}

	if err := r.Coverage.Exclude(exclude); err != nil {
		return errors.Join(errs, err)
	}

	return nil
}

func (r *Report) MeasureCodeToTestRatio(root string, code, test []string) error {
	ratio, err := ratio.Measure(root, code, test)
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
	repo, err := gh.Parse(r.Repository)
	if err != nil {
		return err
	}
	g, err := gh.New()
	if err != nil {
		return err
	}
	if len(stepNames) > 0 {
		var steps []gh.Step
		for _, n := range stepNames {
			s, err := g.FetchStepsByName(ctx, repo.Owner, repo.Repo, n)
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

	var steps []gh.Step
	for _, path := range r.covPaths {
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}
		jobID, err := g.DetectCurrentJobID(ctx, repo.Owner, repo.Repo)
		if err != nil {
			return err
		}
		s, err := g.FetchStepByTime(ctx, repo.Owner, repo.Repo, jobID, fi.ModTime())
		if err != nil {
			return err
		}
		steps = append(steps, s)
	}

	if len(steps) == 0 {
		return errors.New("could not detect test steps")
	}

	d := mergeExecutionTimes(steps)
	t := float64(d)
	r.TestExecutionTime = &t
	return nil
}

// CollectCustomMetrics collects custom metrics from env.
func (r *Report) CollectCustomMetrics() error {
	const envPrefix = "OCTOCOV_CUSTOM_METRICS_"
	var envs [][]string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, envPrefix) {
			continue
		}
		kv := strings.SplitN(e, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.TrimSpace(kv[1])
		envs = append(envs, []string{k, v})
	}
	// Sort by key
	sort.Slice(envs, func(i, j int) bool {
		return envs[i][0] < envs[j][0]
	})
	for _, env := range envs {
		v := env[1]
		b, err := os.ReadFile(v)
		if err != nil {
			return err
		}
		var sets []*CustomMetricSet
		if err := json.Unmarshal(b, &sets); err != nil {
			set := &CustomMetricSet{}
			if err := json.Unmarshal(b, set); err != nil {
				return err
			}
			sets = append(sets, set)
		}
		for _, set := range sets {
			set.report = r
			// Validate
			if err := set.Validate(); err != nil {
				return err
			}
			if len(set.Metrics) != len(lo.UniqBy(set.Metrics, func(m *CustomMetric) string {
				return m.Key
			})) {
				return fmt.Errorf("key of metrics must be unique: %s", lo.Map(set.Metrics, func(m *CustomMetric, _ int) string {
					return m.Key
				}))
			}
			r.CustomMetrics = append(r.CustomMetrics, set)
		}
	}

	// Validate
	if len(r.CustomMetrics) != len(lo.UniqBy(r.CustomMetrics, func(s *CustomMetricSet) string {
		return s.Key
	})) {
		return fmt.Errorf("key of custom metrics must be unique: %s", lo.Map(r.CustomMetrics, func(s *CustomMetricSet, _ int) string {
			return s.Key
		}))
	}

	return nil
}

func (r *Report) CoveragePercent() float64 {
	if r == nil || r.Coverage == nil || r.Coverage.Total == 0 {
		return 0.0
	}
	return float64(r.Coverage.Covered) / float64(r.Coverage.Total) * 100
}

func (r *Report) CodeToTestRatioRatio() float64 {
	if r == nil || r.CodeToTestRatio == nil || r.CodeToTestRatio.Code == 0 {
		return 0.0
	}
	return float64(r.CodeToTestRatio.Test) / float64(r.CodeToTestRatio.Code)
}

func (r *Report) TestExecutionTimeNano() float64 {
	if r == nil || r.TestExecutionTime == nil {
		return 0.0
	}
	return *r.TestExecutionTime
}

func (r *Report) Validate() error {
	if r.Repository == "" {
		return fmt.Errorf("coverage report %q (env %s) is not set", "repository", "GITHUB_REPOSITORY")
	}
	if r.Ref == "" {
		return fmt.Errorf("coverage report %q (env %s) is not set", "ref", "GITHUB_REF")
	}
	if r.Commit == "" {
		return fmt.Errorf("coverage report %q (env %s) is not set", "commit", "GITHUB_SHA")
	}

	if len(r.CustomMetrics) != len(lo.UniqBy(r.CustomMetrics, func(s *CustomMetricSet) string {
		return s.Key
	})) {
		return fmt.Errorf("key of custom metrics must be unique: %s", lo.Map(r.CustomMetrics, func(s *CustomMetricSet, _ int) string {
			return s.Key
		}))
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
	if r.IsMeasuredCoverage() {
		d.Coverage = r.Coverage.Compare(r2.Coverage)
	}
	if r.IsMeasuredCodeToTestRatio() {
		d.CodeToTestRatio = r.CodeToTestRatio.Compare(r2.CodeToTestRatio)
	}
	if r.IsMeasuredTestExecutionTime() {
		dt := &DiffTestExecutionTime{
			A:                  r.TestExecutionTime,
			B:                  r2.TestExecutionTime,
			TestExecutionTimeA: r.TestExecutionTime,
			TestExecutionTimeB: r2.TestExecutionTime,
		}
		var t1, t2 float64
		t1 = r.TestExecutionTimeNano()
		if r2.TestExecutionTime != nil {
			t2 = r2.TestExecutionTimeNano()
		}
		dt.Diff = t1 - t2
		d.TestExecutionTime = dt
	}
	if r.IsCollectedCustomMetrics() {
		for _, set := range r.CustomMetrics {
			set2 := r2.findCustomMetricSetByKey(set.Key)
			d.CustomMetrics = append(d.CustomMetrics, set.Compare(set2))
		}
	}
	return d
}

func (r *Report) CustomMetricsAcceptable(cr config.Reporter) error {
	if cr == nil {
		return nil
	}
	rPrev, ok := cr.(*Report)
	if !ok {
		return fmt.Errorf("type assertion error: %T to *Report", cr)
	}
	if rPrev == nil || len(rPrev.CustomMetrics) == 0 {
		return nil
	}
	var errs []error
	for _, set := range r.CustomMetrics {
		setPrev, ok := lo.Find(rPrev.CustomMetrics, func(s *CustomMetricSet) bool {
			return s.Key == set.Key
		})
		if !ok {
			continue
		}
		current := map[string]float64{}
		prev := map[string]float64{}
		diff := map[string]float64{}
		for _, m := range set.Metrics {
			current[m.Key] = m.Value
			mPrev, ok := lo.Find(setPrev.Metrics, func(mm *CustomMetric) bool {
				return mm.Key == m.Key
			})
			var prevVal float64
			if ok {
				prevVal = mPrev.Value
			}
			prev[m.Key] = prevVal
			diff[m.Key] = current[m.Key] - prev[m.Key]
		}
		variables := map[string]any{
			"current": current,
			"prev":    prev,
			"diff":    diff,
		}
		for _, cond := range set.Acceptables {
			if cond == "" {
				continue
			}
			ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
			if err != nil {
				errs = append(errs, err)
			}
			tf, okk := ok.(bool)
			if !okk {
				errs = append(errs, fmt.Errorf("invalid condition: %q", cond))
			}
			if !tf {
				errs = append(errs, fmt.Errorf("not acceptable condition: %q", cond))
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (r *Report) findCustomMetricSetByKey(key string) *CustomMetricSet {
	for _, set := range r.CustomMetrics {
		if set.Key == key {
			return set
		}
	}
	return nil
}

func (r *Report) convertFormat(v any) string {
	if r.opts != nil && r.opts.Locale != nil {
		p := message.NewPrinter(*r.opts.Locale)
		return p.Sprint(number.Decimal(v))
	}

	switch vv := v.(type) {
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", vv)
	case float64:
		if isInt(vv) {
			return fmt.Sprintf("%d", int(vv))
		}
		return fmt.Sprintf("%.1f", floor1(vv))
	default:
		panic(fmt.Errorf("convert format error .Unknown type:%v", vv)) //nostyle:dontpanic
	}
}

func makeHeadTitle(ref, commit string, covPaths []string) string {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	if strings.HasPrefix(ref, "refs/pull/") {
		ref = strings.Replace(strings.TrimSuffix(strings.TrimSuffix(ref, "/head"), "/merge"), "refs/pull/", "#", 1)
	}
	if len(commit) > 7 {
		commit = commit[:7]
	} else {
		commit = "-"
	}
	if ref == "" {
		return strings.Join(covPaths, ", ")
	}
	return fmt.Sprintf("%s (%s)", ref, commit)
}

func makeHeadTitleWithLink(ref, commit string, covPaths []string) string {
	var (
		refLink    string
		commitLink string
	)
	repoURL := fmt.Sprintf("%s/%s", os.Getenv("GITHUB_SERVER_URL"), os.Getenv("GITHUB_REPOSITORY"))
	switch {
	case strings.HasPrefix(ref, "refs/heads/"):
		branch := strings.TrimPrefix(ref, "refs/heads/")
		refLink = fmt.Sprintf("[%s](%s/tree/%s)", branch, repoURL, branch)
	case strings.HasPrefix(ref, "refs/pull/"):
		n := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSuffix(ref, "/head"), "/merge"), "refs/pull/")
		refLink = fmt.Sprintf("[#%s](%s/pull/%s)", n, repoURL, n)
	default:
		refLink = ref
	}
	if len(commit) > 7 {
		commitLink = fmt.Sprintf("[%s](%s/commit/%s)", commit[:7], repoURL, commit)
	} else {
		commitLink = "-"
	}
	if ref == "" {
		return strings.Join(covPaths, ", ")
	}
	return fmt.Sprintf("%s (%s)", refLink, commitLink)
}

type timePoint struct {
	t time.Time
	c int
}

func mergeExecutionTimes(steps []gh.Step) time.Duration {
	var timePoints []timePoint
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
	} else {
		log.Printf("parse as Go coverage: %s", err)
	}
	// lcov
	if cov, rp, err := coverage.NewLcov().ParseReport(path); err == nil {
		return cov, rp, nil
	} else {
		log.Printf("parse as LCOV: %s", err)
	}
	// simplecov
	if cov, rp, err := coverage.NewSimplecov().ParseReport(path); err == nil {
		return cov, rp, nil
	} else {
		log.Printf("parse as SimpleCov: %s", err)
	}
	// clover
	if cov, rp, err := coverage.NewClover().ParseReport(path); err == nil {
		return cov, rp, nil
	} else {
		log.Printf("parse as Clover: %s", err)
	}
	// cobertura
	if cov, rp, err := coverage.NewCobertura().ParseReport(path); err == nil {
		return cov, rp, nil
	} else {
		log.Printf("parse as Cobertura: %s", err)
	}
	// jacoco
	if cov, rp, err := coverage.NewJacoco().ParseReport(path); err == nil {
		return cov, rp, nil
	} else {
		log.Printf("parse as JaCoCo: %s", err)
	}

	msg := fmt.Sprintf("parsable coverage report not found: %s", path)
	log.Println(msg)

	return nil, "", errors.New(msg)
}

// floor1 round down to one decimal place.
func floor1(v float64) float64 {
	return math.Floor(v*10) / 10
}
