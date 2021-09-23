package config

import (
	"errors"
	"fmt"
	"os"
)

type ConfigReport struct {
	If         string   `yaml:"if,omitempty"`
	Path       string   `yaml:"path,omitempty"`
	Datastores []string `yaml:"datastores,omitempty"`
}

func (c *Config) ReportConfigReady() bool {
	if c.Report == nil {
		return false
	}
	if !c.ReportConfigTargetReady() {
		_, _ = fmt.Fprintln(os.Stderr, "Skip storing the report: report.datastores and report.path are not set")
		return false
	}
	ok, err := CheckIf(c.Report.If)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: %v\n", err)
		return false
	}
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: the condition in the `if` section is not met (%s)\n", c.Report.If)
		return false
	}
	return true
}

func (c *Config) ReportConfigTargetReady() bool {
	return (c.Report != nil && (c.Report.Path != "" || len(c.Report.Datastores) > 0))
}

func (c *Config) BuildReportConfig() error {
	if c.Report.Path == "" && len(c.Report.Datastores) == 0 {
		return errors.New("report.datastores and report.path are not set")
	}
	return nil
}
