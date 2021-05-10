package central

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
)

const defaultHost = "https://github.com"

//go:embed index.md.tmpl
var indexTmpl []byte

type Central struct {
	config  *config.Config
	reports []*report.Report
}

func New(c *config.Config) *Central {
	return &Central{
		config:  c,
		reports: []*report.Report{},
	}
}

func (c *Central) Generate() error {
	// collect reports
	if err := c.collectReports(); err != nil {
		return err
	}

	// generate badges
	if err := c.generateBadges(); err != nil {
		return err
	}

	// render index
	p := c.config.Central.Root
	fi, err := os.Stat(c.config.Central.Root)
	if err == nil && fi.IsDir() {
		p = filepath.Join(c.config.Central.Root, "README.md")
	}
	i, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
	if err != nil {
		return err
	}
	if err := c.renderIndex(i); err != nil {
		return err
	}

	return nil
}

func (c *Central) collectReports() error {
	rsMap := map[string]*report.Report{}

	// collect reports
	if err := filepath.Walk(c.config.Central.Reports, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".json") {
			return nil
		}
		r := report.New()
		b, err := ioutil.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil
		}
		if err := json.Unmarshal(b, r); err != nil {
			return nil
		}
		current, ok := rsMap[r.Repository]
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "Collect report of %s\n", r.Repository)
			rsMap[r.Repository] = r
			return nil
		}
		if current.Timestamp.UnixNano() < r.Timestamp.UnixNano() {
			rsMap[r.Repository] = r
		}
		return nil
	}); err != nil {
		return err
	}

	for _, r := range rsMap {
		c.reports = append(c.reports, r)
	}
	sort.Slice(c.reports, func(i, j int) bool { return c.reports[i].Repository < c.reports[j].Repository })
	return nil
}

func (c *Central) generateBadges() error {
	for _, r := range c.reports {
		cp := r.CoveragePercent()
		err := os.MkdirAll(filepath.Join(c.config.Central.Badges, r.Repository), 0755) // #nosec
		if err != nil {
			return err
		}
		out, err := os.OpenFile(filepath.Join(c.config.Central.Badges, r.Repository, "coverage.svg"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
		if err != nil {
			return err
		}
		b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
		b.ValueColor = c.config.CoverageColor(cp)
		if err := b.Render(out); err != nil {
			return err
		}
	}
	return nil
}

func (c *Central) renderIndex(wr io.Writer) error {
	tmpl := template.Must(template.New("index").Funcs(funcs()).Parse(string(indexTmpl)))
	host := os.Getenv("GITHUB_SERVER_URL")
	if host == "" {
		host = defaultHost
	}

	root := c.config.Central.Root
	if strings.HasSuffix(root, ".md") {
		root = filepath.Dir(c.config.Central.Root)
	}

	badgesRel, err := filepath.Rel(root, c.config.Central.Badges)
	if err != nil {
		return err
	}

	d := map[string]interface{}{
		"Host":      host,
		"Reports":   c.reports,
		"BadgesRel": badgesRel,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}

	return nil
}

func funcs() map[string]interface{} {
	return template.FuncMap{
		"coverage": func(r *report.Report) string {
			return fmt.Sprintf("%.1f%%", r.CoveragePercent())
		},
	}
}
