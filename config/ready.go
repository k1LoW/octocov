package config

import (
	"errors"
	"fmt"
	"os"
)

func (c *Config) CoverageConfigReady() error {
	if c.Coverage == nil {
		return errors.New("coverage: not set")
	}
	if c.Coverage.Path == "" {
		return errors.New("coverage.path: not set")
	}
	return nil
}

func (c *Config) CodeToTestRatioConfigReady() error {
	if c.CodeToTestRatio == nil {
		return errors.New("codeToTestRatio: not set")
	}
	if len(c.CodeToTestRatio.Test) == 0 {
		return errors.New("codeToTestRatio.test: not set")
	}
	return nil
}

func (c *Config) TestExecutionTimeConfigReady() error {
	if c.TestExecutionTime == nil {
		return errors.New("testExecutionTime: not set")
	}
	if err := c.CoverageConfigReady(); err != nil && len(c.TestExecutionTime.Steps) == 0 {
		return err
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
	if c.Comment == nil || !c.Comment.Enable {
		return false
	}
	return true
}

func (c *Config) CoverageBadgeConfigReady() error {
	if err := c.CoverageConfigReady(); err != nil {
		return err
	}
	if c.Coverage.Badge.Path != "" {
		return errors.New("coverage.badge.path: not set")
	}
	return nil
}

func (c *Config) CodeToTestRatioBadgeConfigReady() error {
	if err := c.CodeToTestRatioConfigReady(); err != nil {
		return err
	}
	if c.CodeToTestRatio.Badge.Path != "" {
		return errors.New("codeToTestRatio.badge.path: not set")
	}
	return nil
}

func (c *Config) TestExecutionTimeBadgeConfigReady() error {
	if err := c.TestExecutionTimeConfigReady(); err != nil {
		return err
	}
	if c.TestExecutionTime.Badge.Path != "" {
		return errors.New("testExecutionTime.badge.path: not set")
	}
	return nil
}

func (c *Config) CentralConfigReady() error {
	if c.Central == nil {
		return errors.New("central: not set")
	}
	if !c.Central.Enable {
		return errors.New("central.enable: is false")
	}
	if c.Repository == "" {
		return errors.New("repository: not set (or env GITHUB_REPOSITORY is not set)")
	}
	if len(c.Central.Reports.Datastores) == 0 {
		return errors.New("central.reports.datastores is not set")
	}
	return nil
}

func (c *Config) CentralPushConfigReady() error {
	if err := c.CentralConfigReady(); err != nil {
		return err
	}
	if !c.Central.Push.Enable {
		return errors.New("central.puth.enable: is false")
	}
	if c.GitRoot == "" {
		return errors.New("failed to traverse the Git root path")
	}
	ok, err := CheckIf(c.Central.Push.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)\n", c.Push.If)
	}
	return nil
}

func (c *Config) DiffConfigReady() bool {
	return (c.Diff != nil && (c.Diff.Path != "" || len(c.Diff.Datastores) > 0))
}

func (c *Config) ReportConfigReady() error {
	if c.Report == nil {
		return errors.New("report: not set")
	}
	if !c.ReportConfigTargetReady() {
		return errors.New("report.datastores and report.path are not set")
	}
	ok, err := CheckIf(c.Report.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)\n", c.Report.If)
	}
	return nil
}

func (c *Config) ReportConfigTargetReady() bool {
	return (c.Report != nil && (c.Report.Path != "" || len(c.Report.Datastores) > 0))
}
