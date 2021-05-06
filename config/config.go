package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
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
	Repository string          `yaml:"repository"`
	Coverage   *ConfigCoverage `yaml:"coverage"`
	// CodeToTestRatio *ConfigCodeToTestRatio `yaml:"codeToTestRatio,omitempty"`
	Datastore *ConfigDatastore `yaml:"datastore,omitempty"`
	Central   *ConfigCentral   `yaml:"central,omitempty"`
	// current directory
	root string
	// config file path
	path string
}

type ConfigCoverage struct {
	Path       string `yaml:"path,omitempty"`
	Badge      string `yaml:"badge,omitempty"`
	Acceptable string `yaml:"acceptable,omitempty"`
}

// type ConfigCodeToTestRatio struct {
// 	Enable bool `yaml:"enable"`
//  Badge string `yaml:"badge"`
// }

type ConfigDatastore struct {
	Github *ConfigDatastoreGithub `yaml:"github,omitempty"`
}

type ConfigDatastoreGithub struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

type ConfigCentral struct {
	Enable  bool   `yaml:"enable"`
	Reports string `yaml:"reports"`
	Badges  string `yaml:"badges"`
	Root    string `yaml:"root"`
}

func New() *Config {
	p, _ := os.Getwd()
	return &Config{
		Coverage: &ConfigCoverage{},
		root:     p,
	}
}

func (c *Config) Load(path string) error {
	if path == "" {
		for _, p := range DefaultConfigFilePaths {
			if f, err := os.Stat(filepath.Join(c.root, p)); err == nil && !f.IsDir() {
				if path != "" {
					return fmt.Errorf("duplicate config file [%s, %s]", path, p)
				}
				path = p
			}
		}
	}
	if path == "" {
		c.Coverage.Path = c.root
		return nil
	}
	c.path = path
	buf, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(buf, c); err != nil {
		return err
	}
	if c.Coverage.Path == "" {
		c.Coverage.Path = filepath.Dir(path)
	}
	return nil
}

func (c *Config) Loaded() bool {
	return c.path != ""
}

func (c *Config) Build() {
	c.Repository = os.ExpandEnv(c.Repository)
	if c.Repository == "" {
		c.Repository = os.Getenv("GITHUB_REPOSITORY")
	}
	if c.Datastore != nil && c.Datastore.Github != nil {
		c.Datastore.Github.Repository = os.ExpandEnv(c.Datastore.Github.Repository)
		c.Datastore.Github.Branch = os.ExpandEnv(c.Datastore.Github.Branch)
		c.Datastore.Github.Path = os.ExpandEnv(c.Datastore.Github.Path)
	}
	if c.Coverage != nil {
		c.Coverage.Badge = os.ExpandEnv(c.Coverage.Badge)
	}
	if c.Central != nil {
		c.Central.Root = os.ExpandEnv(c.Central.Root)
		c.Central.Reports = os.ExpandEnv(c.Central.Reports)
		c.Central.Badges = os.ExpandEnv(c.Central.Badges)
	}
}

func (c *Config) DatastoreConfigReady() bool {
	return c.Datastore != nil
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

func (c *Config) BadgeConfigReady() bool {
	return c.Coverage.Badge != ""
}

func (c *Config) Accepptable(r *report.Report) error {
	if c.Coverage.Acceptable == "" {
		return nil
	}
	a, err := strconv.ParseFloat(strings.TrimSuffix(c.Coverage.Acceptable, "%"), 64)
	if err != nil {
		return err
	}

	if r.CoveragePercent() < a {
		return fmt.Errorf("code coverage is %.1f%%, which is below the accepted %.1f%%", r.CoveragePercent(), a)
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
