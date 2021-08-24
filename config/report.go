package config

import (
	"errors"
	"fmt"
	"os"
)

type ConfigReport struct {
	If         string   `yaml:"if,omitempty"`
	Datastores []string `yaml:"datastores,omitempty"`
}

func (c *Config) ReportConfigReady() bool {
	if c.Report == nil {
		return false
	}
	if !c.ReportConfigDatastoresReady() {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: report.datastores is not set %v\n", c.Report.Datastores)
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

func (c *Config) ReportConfigDatastoresReady() bool {
	if c.Report == nil {
		return false
	}
	return len(c.Report.Datastores) > 0
}

func (c *Config) BuildReportConfig() error {
	if len(c.Report.Datastores) == 0 {
		return errors.New("report.datastores is not set")
	}
	return nil
}
