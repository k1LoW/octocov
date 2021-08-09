package config

func (c *Config) DatastoreConfigReady() bool {
	return c.Datastore != nil
}
