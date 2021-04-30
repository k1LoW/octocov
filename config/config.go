package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/k1LoW/octocov/report"
)

const defaultBranch = "main"
const defaultReportDir = "report"

var DefaultConfigFilePaths = []string{".octocov.yml", "octocov.yml"}

type Config struct {
	Repository string         `yaml:"repository"`
	Coverage   ConfigCoverage `yaml:"coverage,omitempty"`
	// CodeToTestRatio ConfigCodeToTestRatio `yaml:"codeToTestRatio,omitempty"`
	Report ConfigReport `yaml:"report,omitempty"`
}

type ConfigCoverage struct {
	Path  string `yaml:"path"`
	Badge string `yaml:"badge"`
}

// type ConfigCodeToTestRatio struct {
// 	Enable bool `yaml:"enable"`
//  Badge string `yaml:"badge"`
// }

type ConfigReport struct {
	Github ConfigReportGithub `yaml:"github,omitempty"`
}

type ConfigReportGithub struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

func New() *Config {
	return &Config{}
}

func (c *Config) Load(path string) error {
	if path == "" {
		for _, p := range DefaultConfigFilePaths {
			if f, err := os.Stat(p); err == nil && !f.IsDir() {
				if path != "" {
					return fmt.Errorf("duplicate config file [%s, %s]", path, p)
				}
				path = p
			}
		}
	}
	if path == "" {
		p, err := os.Getwd()
		if err != nil {
			return err
		}
		c.Coverage.Path = p
		return nil
	}
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

func (c *Config) Build(r *report.Report) {
	if r != nil && c.Repository == "" {
		c.Repository = r.Repository
	}
	c.Report.Github.Repository = os.ExpandEnv(c.Report.Github.Repository)
	c.Report.Github.Branch = os.ExpandEnv(c.Report.Github.Branch)
	if c.Report.Github.Branch == "" {
		c.Report.Github.Branch = defaultBranch
	}
	c.Report.Github.Path = os.ExpandEnv(c.Report.Github.Path)
	if c.Report.Github.Path == "" && c.Repository != "" {
		c.Report.Github.Path = fmt.Sprintf("%s/%s.json", defaultReportDir, c.Repository)
	}
}

func (c *Config) ValidateReportConfig() error {
	if c.Report.Github.Repository == "" {
		return errors.New("report.github.repository not set")
	}
	if strings.Count(c.Report.Github.Repository, "/") != 1 {
		return errors.New("report.github.repository should be 'owner/repo'")
	}
	if c.Report.Github.Branch == "" {
		return errors.New("report.github.branch not set")
	}
	if c.Report.Github.Path == "" {
		return errors.New("report.github.path not set")
	}
	return nil
}
