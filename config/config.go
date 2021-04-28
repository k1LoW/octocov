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
const defaultPushDir = "report"

var DefaultConfigFilePaths = []string{".octocov.yml", "octocov.yml"}

type Config struct {
	Coverage ConfigCoverage `yaml:"coverage,omitempty"`
	Report   ConfigReport   `yaml:"report,omitempty"`
	Push     ConfigPush     `yaml:"push,omitempty"`
	Badge    ConfigBadge    `yaml:"badge,omitempty"`

	root string
}

type ConfigCoverage struct {
	Path string `yaml:"path"`
}

type ConfigReport struct {
	Repository string `yaml:"repository"`
}

type ConfigPush struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

type ConfigBadge struct {
	Path string `yaml:"path"`
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
	buf, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(buf, c); err != nil {
		return err
	}
	return nil
}

func (c *Config) SetReport(r *report.Report) error {
	if c.Report.Repository == "" {
		c.Report.Repository = r.Repository
	}
	return nil
}

func (c *Config) BuildPushConfig() error {
	c.Push.Repository = os.ExpandEnv(c.Push.Repository)
	if c.Push.Repository == "" {
		return errors.New("push.repository not set")
	}
	if strings.Count(c.Push.Repository, "/") != 1 {
		return errors.New("push.repository should be 'owner/repo'")
	}

	c.Push.Branch = os.ExpandEnv(c.Push.Branch)
	if c.Push.Branch == "" {
		c.Push.Branch = defaultBranch
	}

	c.Push.Path = os.ExpandEnv(c.Push.Path)
	if c.Push.Path == "" {
		if c.Report.Repository == "" {
			return errors.New("report.repository not set")
		}
		c.Push.Path = fmt.Sprintf("%s/%s.json", defaultPushDir, c.Report.Repository)
	}

	return nil
}
