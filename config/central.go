package config

import "errors"

func (c *Config) CentralConfigReady() bool {
	return (c.Central != nil && c.Central.Enable)
}

func (c *Config) BuildCentralConfig() error {
	if c.Central == nil {
		return errors.New("central: not set")
	}
	if c.Central.Root == "" {
		c.Central.Root = "."
	}
	if c.Central.Reports == "" {
		c.Central.Reports = defaultReportsDir
	}
	if c.Central.Badges == "" {
		c.Central.Badges = defaultBadgesDir
	}
	return nil
}
