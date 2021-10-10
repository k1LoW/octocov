package config

type ConfigReport struct {
	If         string   `yaml:"if,omitempty"`
	Path       string   `yaml:"path,omitempty"`
	Datastores []string `yaml:"datastores,omitempty"`
}
