package config

func (c *Config) DatastoreConfigReady() bool {
	if c.Datastore == nil {
		return false
	}
	return true
}
