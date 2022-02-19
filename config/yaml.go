package config

import (
	"regexp"

	"github.com/goccy/go-yaml"
)

var commentRe = regexp.MustCompile(`(?m)^comment:`)
var pushRe = regexp.MustCompile(`(?m)^push:`)
var centralPushRe = regexp.MustCompile(`(?m)^\s+push:`)

func (c *Config) UnmarshalYAML(data []byte) error {
	s := struct {
		Repository        string                   `yaml:"repository"`
		Coverage          *ConfigCoverage          `yaml:"coverage"`
		CodeToTestRatio   *ConfigCodeToTestRatio   `yaml:"codeToTestRatio,omitempty"`
		TestExecutionTime *ConfigTestExecutionTime `yaml:"testExecutionTime,omitempty"`
		Report            *ConfigReport            `yaml:"report,omitempty"`
		Central           *ConfigCentral           `yaml:"central,omitempty"`
		Push              interface{}              `yaml:"push,omitempty"`
		Comment           interface{}              `yaml:"comment,omitempty"`
		Diff              *ConfigDiff              `yaml:"diff,omitempty"`
	}{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	c.Repository = s.Repository
	c.Coverage = s.Coverage
	c.CodeToTestRatio = s.CodeToTestRatio
	c.TestExecutionTime = s.TestExecutionTime
	c.Report = s.Report
	c.Central = s.Central
	c.Diff = s.Diff

	switch v := s.Comment.(type) {
	case nil:
		if commentRe.Match(data) {
			c.Comment = &ConfigComment{}
		}
	case *ConfigComment:
		c.Comment = v
	}

	switch v := s.Push.(type) {
	case nil:
		if pushRe.Match(data) {
			c.Push = &ConfigPush{}
		}
	case *ConfigPush:
		c.Push = v
	}

	return nil
}

func (c *ConfigCentral) UnmarshalYAML(data []byte) error {
	s := struct {
		Root    string               `yaml:"root"`
		Reports ConfigCentralReports `yaml:"reports"`
		Badges  ConfigCentralBadges  `yaml:"badges"`
		Push    interface{}          `yaml:"push,omitempty"`
		If      string               `yaml:"if,omitempty"`
	}{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	c.Root = s.Root
	c.Reports = s.Reports
	c.Badges = s.Badges
	c.If = s.If

	switch v := s.Push.(type) {
	case nil:
		if centralPushRe.Match(data) {
			c.Push = &ConfigPush{}
		}
	case *ConfigPush:
		c.Push = v
	}

	return nil
}
