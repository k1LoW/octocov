package config

func (c *Config) DiffConfigReady() bool {
	return (c.Diff != nil && (c.Diff.Path != "" || len(c.Diff.Datastores) > 0))
}
