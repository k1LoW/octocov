package config

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/goccy/go-yaml"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/errors"
	"github.com/k1LoW/expand"
	cov "github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal"
	"golang.org/x/text/language"
)

const defaultBadgesDatastore = "local://reports"
const defaultReportsDatastore = "local://reports"
const defaultTimeout = "30sec"
const largeEnoughTime = float64(99 * time.Hour)

var Paths = internal.ConfigPaths

type Config struct {
	Repository        string             `yaml:"repository"`
	Coverage          *Coverage          `yaml:"coverage"`
	CodeToTestRatio   *CodeToTestRatio   `yaml:"codeToTestRatio,omitempty"`
	TestExecutionTime *TestExecutionTime `yaml:"testExecutionTime,omitempty"`
	Report            *Report            `yaml:"report,omitempty"`
	Central           *Central           `yaml:"central,omitempty"`
	Push              *Push              `yaml:"push,omitempty"`
	Comment           *Comment           `yaml:"comment,omitempty"`
	Summary           *Summary           `yaml:"summary,omitempty"`
	Body              *Body              `yaml:"body,omitempty"`
	Diff              *Diff              `yaml:"diff,omitempty"`
	Viewer            *Viewer            `yaml:"viewer,omitempty"`
	Timeout           time.Duration      `yaml:"timeout,omitempty"`
	Locale            *language.Tag      `yaml:"locale,omitempty"`
	GitRoot           string             `yaml:"-"`
	// working directory
	wd string
	// config file path
	path string
	gh   *gh.Gh
}

type Coverage struct {
	Path       string   `yaml:"path,omitempty"`
	Paths      []string `yaml:"paths,omitempty"`
	Exclude    []string `yaml:"exclude,omitempty"`
	Badge      Badge    `yaml:"badge,omitempty"`
	Acceptable string   `yaml:"acceptable,omitempty"`
	If         string   `yaml:"if,omitempty"`
}

var patchVarRe = regexp.MustCompile(`\bpatch\b`)

// AcceptableReferencesPatch reports whether the `coverage.acceptable:` condition
// references the `patch` variable.
func (c *Coverage) AcceptableReferencesPatch() bool {
	if c == nil {
		return false
	}
	return patchVarRe.MatchString(c.Acceptable)
}

type CodeToTestRatio struct {
	Code       []string `yaml:"code"`
	Test       []string `yaml:"test"`
	Badge      Badge    `yaml:"badge,omitempty"`
	Acceptable string   `yaml:"acceptable,omitempty"`
	If         string   `yaml:"if,omitempty"`
}

type TestExecutionTime struct {
	Badge      Badge    `yaml:"badge,omitempty"`
	Acceptable string   `yaml:"acceptable,omitempty"`
	Steps      []string `yaml:"steps,omitempty"`
	If         string   `yaml:"if,omitempty"`
}

type Central struct {
	Root     string         `yaml:"root"`
	Reports  CentralReports `yaml:"reports"`
	Badges   CentralBadges  `yaml:"badges"`
	Push     *Push          `yaml:"push,omitempty"`
	ReReport *Report        `yaml:"reReport,omitempty"`
	If       string         `yaml:"if,omitempty"`
}

type CentralReports struct {
	Datastores []string `yaml:"datastores"`
}

type CentralBadges struct {
	Datastores        []string `yaml:"datastores"`
	Coverage          *Badge   `yaml:"coverage,omitempty"`
	CodeToTestRatio   *Badge   `yaml:"codeToTestRatio,omitempty"`
	TestExecutionTime *Badge   `yaml:"testExecutionTime,omitempty"`
}

type Push struct {
	If      string `yaml:"if,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type Comment struct {
	HideFooterLink bool `yaml:"hideFooterLink"`
	// HideCoverageLink is removed in favor of `viewer: none`, and read only to say so.
	HideCoverageLink bool   `yaml:"hideCoverageLink"`
	ExpandDetails    bool   `yaml:"expandDetails"`
	DeletePrevious   bool   `yaml:"deletePrevious"`
	UpdatePrevious   bool   `yaml:"updatePrevious"`
	Message          string `yaml:"message,omitempty"`
	If               string `yaml:"if,omitempty"`
}

type Summary struct {
	HideFooterLink bool `yaml:"hideFooterLink"`
	// HideCoverageLink is removed in favor of `viewer: none`, and read only to say so.
	HideCoverageLink bool   `yaml:"hideCoverageLink"`
	ExpandDetails    bool   `yaml:"expandDetails"`
	Message          string `yaml:"message,omitempty"`
	If               string `yaml:"if,omitempty"`
}

type Body struct {
	HideFooterLink bool `yaml:"hideFooterLink"`
	// HideCoverageLink is removed in favor of `viewer: none`, and read only to say so.
	HideCoverageLink bool   `yaml:"hideCoverageLink"`
	ExpandDetails    bool   `yaml:"expandDetails"`
	Message          string `yaml:"message,omitempty"`
	If               string `yaml:"if,omitempty"`
}

type Diff struct {
	Path       string   `yaml:"path,omitempty"`
	Datastores []string `yaml:"datastores,omitempty"`
	If         string   `yaml:"if,omitempty"`
}

func New() *Config {
	wd, _ := os.Getwd() //nostyle:handlerrors
	return &Config{
		wd: wd,
	}
}

func (c *Config) Wd() string {
	return c.wd
}

func (c *Config) Setwd(path string) {
	c.wd = path
}

func (c *Config) Load(path string) error {
	path = filepath.FromSlash(path)
	if path == "" {
		for _, p := range Paths {
			if f, err := os.Stat(filepath.Join(c.wd, p)); err == nil && !f.IsDir() {
				if path != "" {
					return fmt.Errorf("duplicate config file [%s, %s]", path, p)
				}
				path = p
			}
		}
	}
	if path == "" {
		return nil
	}
	if filepath.IsAbs(path) {
		c.path = path
	} else {
		c.path = filepath.Join(c.wd, path)
	}
	buf, err := os.ReadFile(filepath.Clean(c.path))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(expand.ExpandenvYAMLBytes(buf), c); err != nil {
		return err
	}
	return nil
}

func (c *Config) Root() string {
	if c.path != "" {
		return filepath.Dir(c.path)
	}
	return c.wd
}

func (c *Config) Loaded() bool {
	return c.path != ""
}

type Reporter interface {
	CoveragePercent() float64
	CodeToTestRatioRatio() float64
	TestExecutionTimeNano() float64
	IsMeasuredTestExecutionTime() bool
	CustomMetricsAcceptable(Reporter) error
}

// Acceptable checks r (and rPrev, for comparison) against the configured acceptable conditions.
// pc is the already measured patch coverage, used for the `patch` variable of
// `coverage.acceptable:`. Pass nil if it could not be measured; `patch` then falls back to
// defaultPatchCoverage, and the condition is still evaluated.
//
// Patch coverage is passed in already measured rather than computed here, because it is derived
// from the block coverages, which the caller shrinks away once the report has been stored.
func (c *Config) Acceptable(r, rPrev Reporter, pc *cov.PatchCoverage) error {
	var errs error
	if err := c.CoverageConfigReady(); err == nil {
		prev := big.NewRat(int64(rPrev.CoveragePercent()*10000), 10000)
		curr := big.NewRat(int64(r.CoveragePercent()*10000), 10000)
		var patch *float64
		if c.Coverage.AcceptableReferencesPatch() {
			patch = buildPatchAcceptableVar(pc)
		}
		if err := coverageAcceptable(curr, prev, c.Coverage.Acceptable, patch); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	if err := c.CodeToTestRatioConfigReady(); err == nil {
		prev := big.NewRat(int64(rPrev.CodeToTestRatioRatio()*10000), 10000)
		curr := big.NewRat(int64(r.CodeToTestRatioRatio()*10000), 10000)
		if err := codeToTestRatioAcceptable(curr, prev, c.CodeToTestRatio.Acceptable); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	if err := c.TestExecutionTimeConfigReady(); err == nil {
		prevVal := largeEnoughTime
		if rPrev.IsMeasuredTestExecutionTime() {
			prevVal = rPrev.TestExecutionTimeNano()
		}
		prev := big.NewRat(int64(prevVal*10000), 10000)
		curr := big.NewRat(int64(r.TestExecutionTimeNano()*10000), 10000)
		if err := testExecutionTimeAcceptable(curr, prev, c.TestExecutionTime.Acceptable); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	if err := r.CustomMetricsAcceptable(rPrev); err != nil {
		errs = errors.Join(errs, err)
	}

	if errs != nil {
		return errs
	}
	return nil
}

var (
	trimPercentRe = regexp.MustCompile(`([\d.]+)%`)
	numberOnlyRe  = regexp.MustCompile(`^\s*[\d]+\.?[\d]*\s*$`)
	compOpRe      = regexp.MustCompile(`^\s*[><=].+$`)

	trimRatioPrefixRe = regexp.MustCompile(`1:([\d.]+)`)
	durationRe        = regexp.MustCompile(`[\d][\d\.\sa-z]*[a-z]`)
)

// normalizeCoverageCond normalizes a condition on a measured coverage into an expression.
func normalizeCoverageCond(cond string) (string, error) {
	// Trim '%'
	return expandCond(trimPercentRe.ReplaceAllString(cond, "$1"), ">="), nil
}

// normalizeCodeToTestRatioCond normalizes a condition on a measured code to test ratio into an expression.
func normalizeCodeToTestRatioCond(cond string) (string, error) {
	// Trim '1:'
	return expandCond(trimRatioPrefixRe.ReplaceAllString(cond, "$1"), ">="), nil
}

// normalizeTestExecutionTimeCond normalizes a condition on a measured test execution time into
// an expression. Durations are replaced with nanoseconds, the unit the measured value carries.
func normalizeTestExecutionTimeCond(cond string) (string, error) {
	for _, m := range durationRe.FindAllString(cond, -1) {
		d, err := duration.Parse(m)
		if err != nil {
			return "", err
		}
		cond = strings.Replace(cond, m, strconv.FormatFloat(float64(d), 'f', -1, 64), 1)
	}
	return expandCond(cond, "<="), nil
}

// expandCond expands a condition written as a threshold alone (`60`) or as a comparison alone
// (`> 60`) into a full expression on `current`.
func expandCond(cond, op string) string {
	switch {
	case numberOnlyRe.MatchString(cond):
		return fmt.Sprintf("current %s %s", op, cond)
	case compOpRe.MatchString(cond):
		return fmt.Sprintf("current %s", cond)
	default:
		return cond
	}
}

// defaultPatchCoverage is substituted for the `patch` variable when patch coverage cannot be
// measured. Missing patch data is a data-level gap, like a missing previous report, so the
// condition is still evaluated with a permissive value rather than skipped: `patch >= 70%` passes,
// and any non-patch part of the condition (e.g. `current >= 80%`) keeps being enforced.
const defaultPatchCoverage = 100.0

// buildPatchAcceptableVar computes the `patch` variable for `coverage.acceptable:`.
// It returns nil if patch coverage could not be measured (no pull request context, or the pull
// request changed no line that the coverage report instruments), in which case
// defaultPatchCoverage is used instead of a zero/undefined value. Reporting why patch coverage
// was not measured is left to the caller, as with the other metrics.
func buildPatchAcceptableVar(pc *cov.PatchCoverage) *float64 {
	if pc == nil || pc.Total == 0 {
		return nil
	}
	rate := pc.Rate()
	return &rate
}

func coverageAcceptable(current, prev *big.Rat, cond string, patch *float64) error {
	if cond == "" {
		return nil
	}
	org := cond
	cond, err := normalizeCoverageCond(cond)
	if err != nil {
		return err
	}

	diff := new(big.Rat).Sub(current, prev)
	diffF, _ := diff.Float64()
	currentF, _ := current.Float64()
	prevF, _ := prev.Float64()
	patchF := defaultPatchCoverage
	if patch != nil {
		patchF = *patch
	}
	variables := map[string]any{
		"current": currentF,
		"prev":    prevF,
		"diff":    diffF,
		"patch":   patchF,
	}
	ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
	if err != nil {
		return err
	}

	tf, okk := ok.(bool)
	if !okk {
		return fmt.Errorf("invalid condition `%s`", cond)
	}
	if !tf {
		// Report the measured patch coverage as well, so that a condition failing on the `patch`
		// term shows the value it failed against. patch == nil means it could not be measured
		// (defaultPatchCoverage was substituted), in which case there is no value to report.
		if patch != nil && patchVarRe.MatchString(org) {
			return fmt.Errorf("code coverage is %.1f%% and patch coverage is %.1f%%. the condition in the `coverage.acceptable:` section is not met (`%s`)", floor1(currentF), floor1(patchF), org)
		}
		return fmt.Errorf("code coverage is %.1f%%. the condition in the `coverage.acceptable:` section is not met (`%s`)", floor1(currentF), org)
	}
	return nil
}

func codeToTestRatioAcceptable(current, prev *big.Rat, cond string) error {
	if cond == "" {
		return nil
	}
	org := cond
	cond, err := normalizeCodeToTestRatioCond(cond)
	if err != nil {
		return err
	}

	diff := new(big.Rat).Sub(current, prev)
	diffF, _ := diff.Float64()
	currentF, _ := current.Float64()
	prevF, _ := prev.Float64()
	variables := map[string]any{
		"current": currentF,
		"prev":    prevF,
		"diff":    diffF,
	}
	ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
	if err != nil {
		return err
	}
	tf, okk := ok.(bool)
	if !okk {
		return fmt.Errorf("invalid condition `%s`", cond)
	}
	if !tf {
		return fmt.Errorf("code to test ratio is 1:%.1f. the condition in the `codeToTestRatio.acceptable:` section is not met (`%s`)", floor1(currentF), org)
	}
	return nil
}

func testExecutionTimeAcceptable(current, prev *big.Rat, cond string) error {
	if cond == "" {
		return nil
	}
	org := cond
	cond, err := normalizeTestExecutionTimeCond(cond)
	if err != nil {
		return err
	}

	diff := new(big.Rat).Sub(current, prev)
	diffF, _ := diff.Float64()
	currentF, _ := current.Float64()
	prevF, _ := prev.Float64()
	variables := map[string]any{
		"current": currentF,
		"prev":    prevF,
		"diff":    diffF,
	}
	ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
	if err != nil {
		return err
	}

	tf, okk := ok.(bool)
	if !okk {
		return fmt.Errorf("invalid condition `%s`", cond)
	}
	if !tf {
		return fmt.Errorf("test execution time is %v. the condition in the `testExecutionTime.acceptable:` section is not met (`%s`)", time.Duration(int64(currentF)), org)
	}
	return nil
}

// CoverageColor returns the color of a measured coverage by the built-in thresholds. It is the
// color of the metric itself rather than of a badge, which is what `octocov ls-files` paints a
// listing with. A badge resolves its own color through Badge.CoverageColor, since `colors:`
// describes one badge rather than what a coverage figure means.
func (c *Config) CoverageColor(cover float64) string {
	return defaultCoverageColor(cover)
}

// CodeToTestRatioColor returns the color of a measured code to test ratio by the built-in thresholds.
func (c *Config) CodeToTestRatioColor(ratio float64) string {
	return defaultCodeToTestRatioColor(ratio)
}

// TestExecutionTimeColor returns the color of a measured test execution time by the built-in thresholds.
func (c *Config) TestExecutionTimeColor(d time.Duration) string {
	return defaultTestExecutionTimeColor(d)
}

func (c *Config) CheckIf(cond string) (bool, error) {
	if cond == "" {
		return true, nil
	}
	variables, err := c.ifVariables()
	if err != nil {
		return false, err
	}
	ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
	if err != nil {
		return false, err
	}
	tf, okk := ok.(bool)
	if !okk {
		return false, fmt.Errorf("invalid condition `%s`", cond)
	}
	return tf, nil
}

// ifVariables returns the variables an `if:` condition is evaluated with.
func (c *Config) ifVariables() (map[string]any, error) {
	e, err := gh.DecodeGitHubEvent()
	if err != nil {
		return nil, err
	}
	if c.Repository == "" {
		return nil, fmt.Errorf("env %s is not set", "GITHUB_REPOSITORY")
	}
	ctx := context.Background()
	repo, err := gh.Parse(c.Repository)
	if err != nil {
		return nil, err
	}
	if c.gh == nil {
		g, err := gh.New()
		if err != nil {
			return nil, err
		}
		c.gh = g
	}
	defaultBranch, err := c.gh.FetchDefaultBranch(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return nil, err
	}
	isDefaultBranch := false
	if b, err := c.gh.DetectCurrentBranch(ctx); err == nil {
		if b == defaultBranch {
			isDefaultBranch = true
		}
	}

	isPullRequest := false
	isDraft := false
	var labels []string
	if n, err := c.gh.DetectCurrentPullRequestNumber(ctx, repo.Owner, repo.Repo); err == nil {
		isPullRequest = true
		pr, err := c.gh.FetchPullRequest(ctx, repo.Owner, repo.Repo, n)
		if err != nil {
			return nil, err
		}
		isDraft = pr.IsDraft
		labels = pr.Labels
	}
	now := time.Now()
	return map[string]any{
		"year":    now.UTC().Year(),
		"month":   now.UTC().Month(),
		"day":     now.UTC().Day(),
		"hour":    now.UTC().Hour(),
		"weekday": int(now.UTC().Weekday()),
		"github": map[string]any{
			"event_name": e.Name,
			"event":      e.Payload,
		},
		"env":               envMap(),
		"is_default_branch": isDefaultBranch,
		"is_pull_request":   isPullRequest,
		"is_draft":          isDraft,
		"labels":            labels,
	}, nil
}

func envMap() map[string]string {
	m := map[string]string{}
	for _, kv := range os.Environ() {
		if !strings.Contains(kv, "=") {
			continue
		}
		parts := strings.SplitN(kv, "=", 2)
		k := parts[0]
		if len(parts) < 2 {
			m[k] = ""
			continue
		}
		m[k] = parts[1]
	}
	return m
}

// floor1 round down to one decimal place.
func floor1(v float64) float64 {
	return math.Floor(v*10) / 10
}
