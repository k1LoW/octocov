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
}

type ConfigCoverage struct {
	Path string `yaml:"path"`
}

type ConfigReport struct {
	Repository string `yaml:"repository"`
}

type ConfigPush struct {
	Github ConfigPushGithub `yaml:"github,omitempty"`
}

type ConfigPushGithub struct {
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
	if r != nil && c.Report.Repository == "" {
		c.Report.Repository = r.Repository
	}
	c.Push.Github.Repository = os.ExpandEnv(c.Push.Github.Repository)
	c.Push.Github.Branch = os.ExpandEnv(c.Push.Github.Branch)
	if c.Push.Github.Branch == "" {
		c.Push.Github.Branch = defaultBranch
	}
	c.Push.Github.Path = os.ExpandEnv(c.Push.Github.Path)
	if c.Push.Github.Path == "" && c.Report.Repository != "" {
		c.Push.Github.Path = fmt.Sprintf("%s/%s.json", defaultPushDir, c.Report.Repository)
	}
}

func (c *Config) ValidatePushConfig() error {
	if c.Push.Github.Repository == "" {
		return errors.New("push.repository not set")
	}
	if strings.Count(c.Push.Github.Repository, "/") != 1 {
		return errors.New("push.repository should be 'owner/repo'")
	}
	if c.Push.Github.Branch == "" {
		return errors.New("push.branch not set")
	}
	if c.Push.Github.Path == "" {
		return errors.New("push.path not set")
	}
	return nil
}
