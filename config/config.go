package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antonmedv/expr"
	"github.com/goccy/go-yaml"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/ghdag/env"
	"github.com/k1LoW/ghdag/runner"
	"github.com/k1LoW/octocov/report"
)

const defaultBranch = "main"
const defaultReportsDir = "reports"
const defaultBadgesDir = "badges"

const (
	// https://github.com/badges/shields/blob/7d452472defa0e0bd71d6443393e522e8457f856/badge-maker/lib/color.js#L8-L12
	green       = "#97CA00"
	yellowgreen = "#A4A61D"
	yellow      = "#DFB317"
	orange      = "#FE7D37"
	red         = "#E05D44"
)

var DefaultConfigFilePaths = []string{".octocov.yml", "octocov.yml"}

type Config struct {
	Repository        string                   `yaml:"repository"`
	Coverage          *ConfigCoverage          `yaml:"coverage"`
	CodeToTestRatio   *ConfigCodeToTestRatio   `yaml:"codeToTestRatio,omitempty"`
	TestExecutionTime *ConfigTestExecutionTime `yaml:"testExecutionTime,omitempty"`
	Datastore         *ConfigDatastore         `yaml:"datastore,omitempty"`
	Central           *ConfigCentral           `yaml:"central,omitempty"`
	Push              *ConfigPush              `yaml:"push,omitempty"`
	Comment           *ConfigComment           `yaml:"comment,omitempty"`
	GitRoot           string                   `yaml:"-"`
	// working directory
	wd string
	// config file path
	path string
}

type ConfigCoverage struct {
	Path       string              `yaml:"path,omitempty"`
	Badge      ConfigCoverageBadge `yaml:"badge,omitempty"`
	Acceptable string              `yaml:"acceptable,omitempty"`
}

type ConfigCoverageBadge struct {
	Path string `yaml:"path,omitempty"`
}

type ConfigCodeToTestRatio struct {
	Code       []string                   `yaml:"code"`
	Test       []string                   `yaml:"test"`
	Badge      ConfigCodeToTestRatioBadge `yaml:"badge,omitempty"`
	Acceptable string                     `yaml:"acceptable,omitempty"`
}

type ConfigCodeToTestRatioBadge struct {
	Path string `yaml:"path,omitempty"`
}

type ConfigTestExecutionTime struct {
	Badge      ConfigTestExecutionTimeBadge `yaml:"badge,omitempty"`
	Acceptable string                       `yaml:"acceptable,omitempty"`
	Steps      []string                     `yaml:"steps,omitempty"`
}

type ConfigTestExecutionTimeBadge struct {
	Path string `yaml:"path,omitempty"`
}

type ConfigDatastore struct {
	If     string                 `yaml:"if,omitempty"`
	Github *ConfigDatastoreGithub `yaml:"github,omitempty"`
}

type ConfigDatastoreGithub struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

type ConfigCentral struct {
	Enable  bool       `yaml:"enable"`
	Root    string     `yaml:"root"`
	Reports string     `yaml:"reports"`
	Badges  string     `yaml:"badges"`
	Push    ConfigPush `yaml:"push"`
}

type ConfigPush struct {
	Enable bool   `yaml:"enable"`
	If     string `yaml:"if,omitempty"`
}

type ConfigComment struct {
	Enable bool `yaml:"enable"`
}

func New() *Config {
	wd, _ := os.Getwd()
	return &Config{
		Coverage: &ConfigCoverage{},
		wd:       wd,
	}
}

func (c *Config) Getwd() string {
	return c.wd
}

func (c *Config) Setwd(path string) {
	c.wd = path
}

func (c *Config) Load(path string) error {
	if path == "" {
		for _, p := range DefaultConfigFilePaths {
			if f, err := os.Stat(filepath.Join(c.wd, p)); err == nil && !f.IsDir() {
				if path != "" {
					return fmt.Errorf("duplicate config file [%s, %s]", path, p)
				}
				path = p
			}
		}
	}
	if path == "" {
		c.Coverage.Path = c.wd
		return nil
	}
	c.path = filepath.Join(c.wd, path)
	buf, err := ioutil.ReadFile(filepath.Clean(c.path))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(buf, c); err != nil {
		return err
	}
	if c.Coverage.Path == "" {
		c.Coverage.Path = filepath.Dir(c.path)
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

func (c *Config) Build() {
	c.Repository = os.ExpandEnv(c.Repository)
	if c.Repository == "" {
		c.Repository = os.Getenv("GITHUB_REPOSITORY")
	}
	gitRoot, _ := traverseGitPath(c.Root())
	c.GitRoot = gitRoot
	if c.Datastore != nil && c.Datastore.Github != nil {
		c.Datastore.Github.Repository = os.ExpandEnv(c.Datastore.Github.Repository)
		c.Datastore.Github.Branch = os.ExpandEnv(c.Datastore.Github.Branch)
		c.Datastore.Github.Path = os.ExpandEnv(c.Datastore.Github.Path)
	}
	if c.Coverage != nil {
		c.Coverage.Badge.Path = os.ExpandEnv(c.Coverage.Badge.Path)
	}
	if c.CodeToTestRatio != nil {
		if c.CodeToTestRatio.Code == nil {
			c.CodeToTestRatio.Code = []string{}
		}
		if c.CodeToTestRatio.Test == nil {
			c.CodeToTestRatio.Test = []string{}
		}
	}
	if c.Central != nil {
		c.Central.Root = os.ExpandEnv(c.Central.Root)
		c.Central.Reports = os.ExpandEnv(c.Central.Reports)
		c.Central.Badges = os.ExpandEnv(c.Central.Badges)
	}
}

func (c *Config) CodeToTestRatioReady() bool {
	if c.CodeToTestRatio == nil {
		return false
	}
	if len(c.CodeToTestRatio.Test) == 0 {
		return false
	}
	return true
}

func (c *Config) DatastoreConfigReady() bool {
	if c.Datastore == nil {
		return false
	}
	ok, err := CheckIf(c.Datastore.If)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: %v\n", err)
		return false
	}
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: the condition in the `if` section is not met (%s)\n", c.Datastore.If)
		return false
	}
	return true
}

func (c *Config) BuildDatastoreConfig() error {
	if c.Datastore.Github == nil {
		return errors.New("datastore.github not set")
	}
	// GitHub
	if c.Datastore.Github.Branch == "" {
		c.Datastore.Github.Branch = defaultBranch
	}
	if c.Datastore.Github.Path == "" && c.Repository != "" {
		c.Datastore.Github.Path = fmt.Sprintf("%s/%s/report.json", defaultReportsDir, c.Repository)
	}
	if c.Datastore.Github.Repository == "" {
		return errors.New("datastore.github.repository not set")
	}
	if strings.Count(c.Datastore.Github.Repository, "/") != 1 {
		return errors.New("datastore.github.repository should be 'owner/repo'")
	}
	if c.Datastore.Github.Branch == "" {
		return errors.New("datastore.github.branch not set")
	}
	if c.Datastore.Github.Path == "" {
		return errors.New("datastore.github.path not set")
	}
	return nil
}

func (c *Config) PushConfigReady() bool {
	if c.Push == nil || !c.Push.Enable || c.GitRoot == "" {
		return false
	}
	ok, err := CheckIf(c.Push.If)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Skip pushing badges: %v\n", err)
		return false
	}
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Skip pushing badges: the condition in the `if` section is not met (%s)\n", c.Push.If)
		return false
	}
	return true
}

func (c *Config) CommentConfigReady() bool {
	if c.Comment == nil || !c.Comment.Enable || !strings.Contains(os.Getenv("GITHUB_REF"), "refs/pull/") {
		return false
	}
	return true
}

func (c *Config) BuildPushConfig() error {
	if c.Push == nil {
		return errors.New("push: not set")
	}
	return nil
}

func (c *Config) CoverageBadgeConfigReady() bool {
	return c.Coverage != nil && c.Coverage.Badge.Path != ""
}

func (c *Config) CodeToTestRatioBadgeConfigReady() bool {
	return c.CodeToTestRatioReady() && c.CodeToTestRatio.Badge.Path != ""
}

func (c *Config) TestExecutionTimeBadgeConfigReady() bool {
	return c.TestExecutionTime != nil && c.TestExecutionTime.Badge.Path != ""
}

func (c *Config) Acceptable(r *report.Report) error {
	if c.Coverage.Acceptable != "" {
		a, err := strconv.ParseFloat(strings.TrimSuffix(c.Coverage.Acceptable, "%"), 64)
		if err != nil {
			return err
		}
		if r.CoveragePercent() < a {
			return fmt.Errorf("code coverage is %.1f%%, which is below the accepted %.1f%%", r.CoveragePercent(), a)
		}
	}

	if c.CodeToTestRatioReady() && c.CodeToTestRatio.Acceptable != "" {
		a, err := strconv.ParseFloat(strings.TrimPrefix(c.CodeToTestRatio.Acceptable, "1:"), 64)
		if err != nil {
			return err
		}
		if r.CodeToTestRatioRatio() < a {
			return fmt.Errorf("code to test ratio is 1:%.1f, which is below the accepted 1:%.1f", r.CodeToTestRatioRatio(), a)
		}
	}

	if r.TestExecutionTime != nil && c.TestExecutionTime != nil && c.TestExecutionTime.Acceptable != "" {
		a, err := duration.Parse(c.TestExecutionTime.Acceptable)
		if err != nil {
			return err
		}
		if *r.TestExecutionTime > float64(a) {
			return fmt.Errorf("test execution time is %v, which is below the accepted %v", time.Duration(*r.TestExecutionTime), a)
		}
	}

	return nil
}

func (c *Config) OwnerRepo() (string, string, error) {
	splitted := strings.Split(c.Repository, "/")
	if len(splitted) != 2 {
		return "", "", errors.New("could not get owner and repo")
	}
	owner := splitted[0]
	repo := splitted[1]
	return owner, repo, nil
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

func CheckIf(cond string) (bool, error) {
	if cond == "" {
		return true, nil
	}
	e, err := runner.DecodeGitHubEvent()
	if err != nil {
		return false, err
	}
	now := time.Now()
	variables := map[string]interface{}{
		"year":    now.UTC().Year(),
		"month":   now.UTC().Month(),
		"day":     now.UTC().Day(),
		"hour":    now.UTC().Hour(),
		"weekday": int(now.UTC().Weekday()),
		"github": map[string]interface{}{
			"event_name": e.Name,
			"event":      e.Payload,
		},
		"env": env.EnvMap(),
	}
	ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
	if err != nil {
		return false, err
	}
	return ok.(bool), nil
}

func traverseGitPath(base string) (string, error) {
	p, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	for {
		fi, err := os.Stat(p)
		if err != nil {
			return "", err
		}
		if !fi.IsDir() {
			p = filepath.Dir(p)
			continue
		}
		gitConfig := filepath.Join(p, ".git", "config")
		if fi, err := os.Stat(gitConfig); err == nil && !fi.IsDir() {
			return p, nil
		}
		if p == "/" {
			break
		}
		p = filepath.Dir(p)
	}
	return "", fmt.Errorf("failed to traverse the Git root path: %s", base)
}
