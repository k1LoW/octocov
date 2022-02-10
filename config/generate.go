package config

import (
	"context"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"strings"
	"text/template"
)

//go:embed template/*
var tmplFS embed.FS

func Generate(ctx context.Context, lang string, wr io.Writer) error {
	tmpl := template.Must(template.ParseFS(tmplFS, "template/.octocov.yml.tmpl"))
	cttr := ""
	if lang != "" {
		b, err := fs.ReadFile(tmplFS, fmt.Sprintf("template/.octocov.%s.yml.tmpl", strings.ToLower(lang)))
		if err == nil {
			cttr = string(b)
		} else {
			log.Print(err)
		}
	}
	d := map[string]interface{}{
		"CodeToTestRatio": cttr,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}
	return nil
}
