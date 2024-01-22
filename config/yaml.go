package config

import (
	"regexp"

	"github.com/goccy/go-yaml"
	"github.com/k1LoW/duration"
	"golang.org/x/text/language"
)

var commentRe = regexp.MustCompile(`(?m)^comment:`)
var pushRe = regexp.MustCompile(`(?m)^push:`)
var centralPushRe = regexp.MustCompile(`(?m)^\s+push:`)

func (c *Config) UnmarshalYAML(data []byte) error {
	s := struct {
		Repository        string             `yaml:"repository"`
		Coverage          *Coverage          `yaml:"coverage"`
		CodeToTestRatio   *CodeToTestRatio   `yaml:"codeToTestRatio,omitempty"`
		TestExecutionTime *TestExecutionTime `yaml:"testExecutionTime,omitempty"`
		Report            *Report            `yaml:"report,omitempty"`
		Central           *Central           `yaml:"central,omitempty"`
		Push              any                `yaml:"push,omitempty"`
		Comment           any                `yaml:"comment,omitempty"`
		Summary           *Summary           `yaml:"summary,omitempty"`
		Body              *Body              `yaml:"body,omitempty"`
		Diff              *Diff              `yaml:"diff,omitempty"`
		Timeout           string             `yaml:"timeout,omitempty"`
		Locale            string             `yaml:"locale,omitempty"`
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
	c.Summary = s.Summary
	c.Body = s.Body
	c.Diff = s.Diff
	if s.Timeout == "" {
		s.Timeout = defaultTimeout
	}
	c.Timeout, err = duration.Parse(s.Timeout)
	if err != nil {
		return err
	}

	switch v := s.Comment.(type) {
	case nil:
		if commentRe.Match(data) {
			c.Comment = &Comment{}
		}
	case map[string]any:
		tmp, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		cc := &Comment{}
		if err := yaml.Unmarshal(tmp, cc); err != nil {
			return err
		}
		c.Comment = cc
	case *Comment:
		c.Comment = v
	}

	switch v := s.Push.(type) {
	case nil:
		if pushRe.Match(data) {
			c.Push = &Push{}
		}
	case map[string]any:
		tmp, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		cp := &Push{}
		if err := yaml.Unmarshal(tmp, cp); err != nil {
			return err
		}
		c.Push = cp
	case *Push:
		c.Push = v
	}

	if s.Locale != "" {
		l, err := language.Parse(s.Locale)
		if err != nil {
			return err
		}

		c.Locale = &l
	}

	return nil
}

func (c *Central) UnmarshalYAML(data []byte) error {
	s := struct {
		Root     string         `yaml:"root"`
		Reports  CentralReports `yaml:"reports"`
		Badges   CentralBadges  `yaml:"badges"`
		Push     any            `yaml:"push,omitempty"`
		ReReport *Report        `yaml:"reReport,omitempty"`
		If       string         `yaml:"if,omitempty"`
	}{}
	err := yaml.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	c.Root = s.Root
	c.Reports = s.Reports
	c.Badges = s.Badges
	c.ReReport = s.ReReport
	c.If = s.If

	switch v := s.Push.(type) {
	case nil:
		if centralPushRe.Match(data) {
			c.Push = &Push{}
		}
	case map[string]any:
		tmp, err := yaml.Marshal(v)
		if err != nil {
			return err
		}
		cp := &Push{}
		if err := yaml.Unmarshal(tmp, cp); err != nil {
			return err
		}
		c.Push = cp
	case *Push:
		c.Push = v
	}

	return nil
}
