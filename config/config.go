package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/antonmedv/expr"
	"github.com/goccy/go-yaml"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/expand"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

const defaultBadgesDir = "badges"
const defaultReportsDatastore = "local://reports"

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
	Report            *ConfigReport            `yaml:"report,omitempty"`
	Central           *ConfigCentral           `yaml:"central,omitempty"`
	Push              *ConfigPush              `yaml:"push,omitempty"`
	Comment           *ConfigComment           `yaml:"comment,omitempty"`
	Diff              *ConfigDiff              `yaml:"diff,omitempty"`
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

type ConfigCentral struct {
	Enable  *bool                `yaml:"enable,omitempty"`
	Root    string               `yaml:"root"`
	Reports ConfigCentralReports `yaml:"reports"`
	Badges  string               `yaml:"badges"`
	Push    *ConfigPush          `yaml:"push"`
}

type ConfigCentralReports struct {
	Datastores []string `yaml:"datastores"`
}

type ConfigPush struct {
	Enable *bool  `yaml:"enable,omitempty"`
	If     string `yaml:"if,omitempty"`
}

type ConfigComment struct {
	Enable         *bool  `yaml:"enable,omitempty"`
	HideFooterLink bool   `yaml:"hideFooterLink"`
	If             string `yaml:"if,omitempty"`
}

type ConfigDiff struct {
	Path       string   `yaml:"path,omitempty"`
	Datastores []string `yaml:"datastores,omitempty"`
	If         string   `yaml:"if,omitempty"`
}

func New() *Config {
	wd, _ := os.Getwd()
	return &Config{
		wd: wd,
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
		return nil
	}
	c.path = filepath.Join(c.wd, path)
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

func (c *Config) Acceptable(r *report.Report) error {
	if err := c.CoverageConfigReady(); err == nil && c.Coverage.Acceptable != "" {
		a, err := strconv.ParseFloat(strings.TrimSuffix(c.Coverage.Acceptable, "%"), 64)
		if err != nil {
			return err
		}
		if r.CoveragePercent() < a {
			return fmt.Errorf("code coverage is %.1f%%, which is below the accepted %.1f%%", r.CoveragePercent(), a)
		}
	}

	if err := c.CodeToTestRatioConfigReady(); err == nil && c.CodeToTestRatio.Acceptable != "" {
		a, err := strconv.ParseFloat(strings.TrimPrefix(c.CodeToTestRatio.Acceptable, "1:"), 64)
		if err != nil {
			return err
		}
		if r.CodeToTestRatioRatio() < a {
			return fmt.Errorf("code to test ratio is 1:%.1f, which is below the accepted 1:%.1f", r.CodeToTestRatioRatio(), a)
		}
	}

	if err := c.TestExecutionTimeConfigReady(); err == nil && r.TestExecutionTime != nil && c.TestExecutionTime.Acceptable != "" {
		a, err := duration.Parse(c.TestExecutionTime.Acceptable)
		if err != nil {
			return err
		}
		if *r.TestExecutionTime > float64(a) {
			return fmt.Errorf("test execution time is %v, which is above the accepted %v", time.Duration(*r.TestExecutionTime), a)
		}
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

func CheckIf(cond string) (bool, error) {
	if cond == "" {
		return true, nil
	}
	e, err := gh.DecodeGitHubEvent()
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
		"env": envMap(),
	}
	ok, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), variables)
	if err != nil {
		return false, err
	}
	return ok.(bool), nil
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
