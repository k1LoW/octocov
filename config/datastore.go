package config

func (c *Config) DatastoreConfigReady() bool {
	// Depracated config
	return c.Datastore != nil
}
