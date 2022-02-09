package config

import (
	"context"
	_ "embed"
	"io"
	"text/template"
)

//go:embed template/.octocov.yml.tmpl
var configTmpl []byte

func (c *Config) Generate(ctx context.Context, wr io.Writer) error {
	tmpl := template.Must(template.New("index").Parse(string(configTmpl)))
	d := map[string]interface{}{
		"CodeToTestRatio": "",
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}
	return nil
}
