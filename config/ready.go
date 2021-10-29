package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal"
)

func (c *Config) CoverageConfigReady() error {
	if c.Coverage == nil {
		return errors.New("coverage: is not set")
	}
	if c.Coverage.Path == "" {
		return errors.New("coverage.path: is not set")
	}
	return nil
}

func (c *Config) CodeToTestRatioConfigReady() error {
	if c.CodeToTestRatio == nil {
		return errors.New("codeToTestRatio: is not set")
	}
	if len(c.CodeToTestRatio.Test) == 0 {
		return errors.New("codeToTestRatio.test: is not set")
	}
	return nil
}

func (c *Config) TestExecutionTimeConfigReady() error {
	if c.TestExecutionTime == nil {
		return errors.New("testExecutionTime: is not set")
	}
	if err := c.CoverageConfigReady(); err != nil && len(c.TestExecutionTime.Steps) == 0 {
		return err
	}
	return nil
}

func (c *Config) PushConfigReady() error {
	if c.Push == nil {
		return errors.New("push: is not set")
	}
	if !internal.IsEnable(c.Push.Enable) {
		return errors.New("push.enable: is false")
	}
	if c.GitRoot == "" {
		return errors.New("failed to traverse the Git root path")
	}
	ok, err := c.CheckIf(c.Push.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)", c.Push.If)
	}
	return nil
}

func (c *Config) CommentConfigReady() error {
	if c.Comment == nil {
		return errors.New("comment: is not set")
	}
	if !internal.IsEnable(c.Comment.Enable) {
		return errors.New("comment.enable: is false")
	}
	if c.Repository == "" {
		return fmt.Errorf("env %s is not set", "GITHUB_REPOSITORY")
	}
	ctx := context.Background()
	repo, err := gh.Parse(c.Repository)
	if err != nil {
		return err
	}
	if c.gh == nil {
		g, err := gh.New()
		if err != nil {
			return err
		}
		c.gh = g
	}
	if _, err := c.gh.DetectCurrentPullRequestNumber(ctx, repo.Owner, repo.Repo); err != nil {
		return err
	}
	ok, err := c.CheckIf(c.Comment.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)", c.Comment.If)
	}
	return nil
}

func (c *Config) CoverageBadgeConfigReady() error {
	if err := c.CoverageConfigReady(); err != nil {
		return err
	}
	if c.Coverage.Badge.Path == "" {
		return errors.New("coverage.badge.path: is not set")
	}
	return nil
}

func (c *Config) CodeToTestRatioBadgeConfigReady() error {
	if err := c.CodeToTestRatioConfigReady(); err != nil {
		return err
	}
	if c.CodeToTestRatio.Badge.Path == "" {
		return errors.New("codeToTestRatio.badge.path: is not set")
	}
	return nil
}

func (c *Config) TestExecutionTimeBadgeConfigReady() error {
	if err := c.TestExecutionTimeConfigReady(); err != nil {
		return err
	}
	if c.TestExecutionTime.Badge.Path == "" {
		return errors.New("testExecutionTime.badge.path: is not set")
	}
	return nil
}

func (c *Config) CentralConfigReady() error {
	if c.Central == nil {
		return errors.New("central: is not set")
	}
	if !internal.IsEnable(c.Central.Enable) {
		return errors.New("central.enable: is false")
	}
	if c.Repository == "" {
		return errors.New("repository: not set (or env GITHUB_REPOSITORY is not set)")
	}
	if len(c.Central.Reports.Datastores) == 0 {
		return errors.New("central.reports.datastores is not set")
	}
	ok, err := c.CheckIf(c.Central.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)", c.Central.If)
	}
	return nil
}

func (c *Config) CentralPushConfigReady() error {
	if err := c.CentralConfigReady(); err != nil {
		return err
	}
	if c.Central.Push == nil {
		return errors.New("central.push: is not set")
	}
	if !internal.IsEnable(c.Central.Push.Enable) {
		return errors.New("central.puth.enable: is false")
	}
	if c.GitRoot == "" {
		return errors.New("failed to traverse the Git root path")
	}
	ok, err := c.CheckIf(c.Central.Push.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)", c.Central.Push.If)
	}
	return nil
}

func (c *Config) DiffConfigReady() error {
	if c.Diff == nil {
		return errors.New("diff: is not set")
	}
	if c.Diff.Path == "" && len(c.Diff.Datastores) == 0 {
		return errors.New("diff.path: and diff.datastores: are not set")
	}
	ok, err := c.CheckIf(c.Diff.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)", c.Diff.If)
	}
	return nil
}

func (c *Config) ReportConfigReady() error {
	if err := c.ReportConfigTargetReady(); err != nil {
		return err
	}
	ok, err := c.CheckIf(c.Report.If)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the condition in the `if` section is not met (%s)", c.Report.If)
	}
	return nil
}

func (c *Config) ReportConfigTargetReady() error {
	if c.Report == nil {
		return errors.New("report: is not set")
	}
	if c.Report.Path == "" && len(c.Report.Datastores) == 0 {
		return errors.New("report.datastores: and report.path: are not set")
	}
	return nil
}
