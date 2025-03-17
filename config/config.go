package config

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/goccy/go-yaml"
	"github.com/hashicorp/go-multierror"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/expand"
	"github.com/k1LoW/octocov/gh"
	"golang.org/x/text/language"
)

const defaultBadgesDatastore = "local://reports"
const defaultReportsDatastore = "local://reports"
const defaultTimeout = "30sec"
const largeEnoughTime = float64(99 * time.Hour)

const (
	// https://github.com/badges/shields/blob/7d452472defa0e0bd71d6443393e522e8457f856/badge-maker/lib/color.js#L8-L12
	green       = "#97CA00"
	yellowgreen = "#A4A61D"
	yellow      = "#DFB317"
	orange      = "#FE7D37"
	red         = "#E05D44"
)

var DefaultPaths = []string{".octocov.yml", "octocov.yml"}

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
	Path       string        `yaml:"path,omitempty"`
	Paths      []string      `yaml:"paths,omitempty"`
	Exclude    []string      `yaml:"exclude,omitempty"`
	Badge      CoverageBadge `yaml:"badge,omitempty"`
	Acceptable string        `yaml:"acceptable,omitempty"`
	If         string        `yaml:"if,omitempty"`
}

type CoverageBadge struct {
	Path string `yaml:"path,omitempty"`
}

type CodeToTestRatio struct {
	Code       []string             `yaml:"code"`
	Test       []string             `yaml:"test"`
	Badge      CodeToTestRatioBadge `yaml:"badge,omitempty"`
	Acceptable string               `yaml:"acceptable,omitempty"`
	If         string               `yaml:"if,omitempty"`
}

type CodeToTestRatioBadge struct {
	Path string `yaml:"path,omitempty"`
}

type TestExecutionTime struct {
	Badge      TestExecutionTimeBadge `yaml:"badge,omitempty"`
	Acceptable string                 `yaml:"acceptable,omitempty"`
	Steps      []string               `yaml:"steps,omitempty"`
	If         string                 `yaml:"if,omitempty"`
}

type TestExecutionTimeBadge struct {
	Path string `yaml:"path,omitempty"`
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
	Datastores []string `yaml:"datastores"`
}

type Push struct {
	If      string `yaml:"if,omitempty"`
	Message string `yaml:"message,omitempty"`
}

type Comment struct {
	HideFooterLink bool   `yaml:"hideFooterLink"`
	DisableDetails bool   `yaml:"disableDetails"`
	DeletePrevious bool   `yaml:"deletePrevious"`
	If             string `yaml:"if,omitempty"`
}

type Summary struct {
	HideFooterLink bool   `yaml:"hideFooterLink"`
	DisableDetails bool   `yaml:"disableDetails"`
	If             string `yaml:"if,omitempty"`
}

type Body struct {
	HideFooterLink bool   `yaml:"hideFooterLink"`
	DisableDetails bool   `yaml:"disableDetails"`
	If             string `yaml:"if,omitempty"`
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
		for _, p := range DefaultPaths {
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
}

func (c *Config) Acceptable(r, rPrev Reporter) error {
	var result *multierror.Error
	if err := c.CoverageConfigReady(); err == nil {
		prev := rPrev.CoveragePercent()
		if err := coverageAcceptable(r.CoveragePercent(), prev, c.Coverage.Acceptable); err != nil {
			result = multierror.Append(result, err)
		}
	}

	if err := c.CodeToTestRatioConfigReady(); err == nil {
		prev := rPrev.CodeToTestRatioRatio()
		if err := codeToTestRatioAcceptable(r.CodeToTestRatioRatio(), prev, c.CodeToTestRatio.Acceptable); err != nil {
			result = multierror.Append(result, err)
		}
	}

	if err := c.TestExecutionTimeConfigReady(); err == nil {
		prev := largeEnoughTime
		if rPrev.IsMeasuredTestExecutionTime() {
			prev = rPrev.TestExecutionTimeNano()
		}

		if err := testExecutionTimeAcceptable(r.TestExecutionTimeNano(), prev, c.TestExecutionTime.Acceptable); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result.ErrorOrNil()
}

var (
	trimPercentRe = regexp.MustCompile(`([\d.]+)%`)
	numberOnlyRe  = regexp.MustCompile(`^\s*[\d]+\.?[\d]*\s*$`)
	compOpRe      = regexp.MustCompile(`^\s*[><=].+$`)

	trimRatioPrefixRe = regexp.MustCompile(`1:([\d.]+)`)
	durationRe        = regexp.MustCompile(`[\d][\d\.\sa-z]*[a-z]`)
)

func coverageAcceptable(current, prev float64, cond string) error {
	if cond == "" {
		return nil
	}
	org := cond
	// Trim '%'
	cond = trimPercentRe.ReplaceAllString(cond, "$1")

	if numberOnlyRe.MatchString(cond) {
		cond = fmt.Sprintf("current >= %s", cond)
	} else if compOpRe.MatchString(cond) {
		cond = fmt.Sprintf("current %s", cond)
	}

	variables := map[string]any{
		"current": current,
		"prev":    prev,
		"diff":    current - prev,
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
		return fmt.Errorf("code coverage is %.1f%%. the condition in the `coverage.acceptable:` section is not met (`%s`)", floor1(current), org)
	}
	return nil
}

func codeToTestRatioAcceptable(current, prev float64, cond string) error {
	if cond == "" {
		return nil
	}
	org := cond
	// Trim '1:'
	cond = trimRatioPrefixRe.ReplaceAllString(cond, "$1")

	if numberOnlyRe.MatchString(cond) {
		cond = fmt.Sprintf("current >= %s", cond)
	} else if compOpRe.MatchString(cond) {
		cond = fmt.Sprintf("current %s", cond)
	}

	variables := map[string]any{
		"current": current,
		"prev":    prev,
		"diff":    current - prev,
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
		return fmt.Errorf("code to test ratio is 1:%.1f. the condition in the `codeToTestRatio.acceptable:` section is not met (`%s`)", floor1(current), org)
	}
	return nil
}

func testExecutionTimeAcceptable(current, prev float64, cond string) error {
	if cond == "" {
		return nil
	}
	org := cond
	matches := durationRe.FindAllString(cond, -1)
	for _, m := range matches {
		d, err := duration.Parse(m)
		if err != nil {
			return err
		}
		cond = strings.Replace(cond, m, strconv.FormatFloat(float64(d), 'f', -1, 64), 1)
	}

	if numberOnlyRe.MatchString(cond) {
		cond = fmt.Sprintf("current <= %s", cond)
	} else if compOpRe.MatchString(cond) {
		cond = fmt.Sprintf("current %s", cond)
	}

	variables := map[string]any{
		"current": current,
		"prev":    prev,
		"diff":    current - prev,
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
		return fmt.Errorf("test execution time is %v. the condition in the `testExecutionTime.acceptable:` section is not met (`%s`)", time.Duration(current), org)
	}
	return nil
}

func (c *Config) CoverageColor(cover float64) string {
	switch {
	case cover >= 80.0:
		return green
	case cover >= 60.0:
		return yellowgreen
	case cover >= 40.0:
		return yellow
	case cover >= 20.0:
		return orange
	default:
		return red
	}
}

func (c *Config) CodeToTestRatioColor(ratio float64) string {
	switch {
	case ratio >= 1.2:
		return green
	case ratio >= 1.0:
		return yellowgreen
	case ratio >= 0.8:
		return yellow
	case ratio >= 0.6:
		return orange
	default:
		return red
	}
}

func (c *Config) TestExecutionTimeColor(d time.Duration) string {
	switch {
	case d < 5*time.Minute:
		return green
	case d < 10*time.Minute:
		return yellowgreen
	case d < 15*time.Minute:
		return yellow
	case d < 20*time.Minute:
		return orange
	default:
		return red
	}
}

func (c *Config) CheckIf(cond string) (bool, error) {
	if cond == "" {
		return true, nil
	}
	e, err := gh.DecodeGitHubEvent()
	if err != nil {
		return false, err
	}
	if c.Repository == "" {
		return false, fmt.Errorf("env %s is not set", "GITHUB_REPOSITORY")
	}
	ctx := context.Background()
	repo, err := gh.Parse(c.Repository)
	if err != nil {
		return false, err
	}
	if c.gh == nil {
		g, err := gh.New()
		if err != nil {
			return false, err
		}
		c.gh = g
	}
	defaultBranch, err := c.gh.FetchDefaultBranch(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return false, err
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
			return false, err
		}
		isDraft = pr.IsDraft
		labels = pr.Labels
	}
	now := time.Now()
	variables := map[string]any{
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
