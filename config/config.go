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
const defaultReportDir = "reports"

var DefaultConfigFilePaths = []string{".octocov.yml", "octocov.yml"}

type Config struct {
	Repository string         `yaml:"repository"`
	Coverage   ConfigCoverage `yaml:"coverage,omitempty"`
	// CodeToTestRatio ConfigCodeToTestRatio `yaml:"codeToTestRatio,omitempty"`
	Datastore ConfigDatastore `yaml:"datastore,omitempty"`
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
	Github ConfigDatastoreGithub `yaml:"github,omitempty"`
}

type ConfigDatastoreGithub struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

func New() *Config {
	p, _ := os.Getwd()
	return &Config{
		root: p,
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
	c.Datastore.Github.Repository = os.ExpandEnv(c.Datastore.Github.Repository)
	c.Datastore.Github.Branch = os.ExpandEnv(c.Datastore.Github.Branch)
	c.Datastore.Github.Path = os.ExpandEnv(c.Datastore.Github.Path)
	c.Coverage.Badge = os.ExpandEnv(c.Coverage.Badge)
}

func (c *Config) DatastoreConfigReady() bool {
	return c.Datastore.Github.Repository != "" || c.Datastore.Github.Branch != "" || c.Datastore.Github.Path != ""
}

func (c *Config) BuildDatastoreConfig() error {
	if c.Datastore.Github.Branch == "" {
		c.Datastore.Github.Branch = defaultBranch
	}
	if c.Datastore.Github.Path == "" && c.Repository != "" {
		c.Datastore.Github.Path = fmt.Sprintf("%s/%s.json", defaultReportDir, c.Repository)
	}
	if c.Datastore.Github.Repository == "" {
		return errors.New("report.github.repository not set")
	}
	if strings.Count(c.Datastore.Github.Repository, "/") != 1 {
		return errors.New("report.github.repository should be 'owner/repo'")
	}
	if c.Datastore.Github.Branch == "" {
		return errors.New("report.github.branch not set")
	}
	if c.Datastore.Github.Path == "" {
		return errors.New("report.github.path not set")
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
