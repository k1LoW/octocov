package central

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
)

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
	if err := c.collectReports(); err != nil {
		return err
	}
	// generate badges
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
			_, _ = fmt.Fprintf(os.Stderr, "collect %s\n", r.Repository)
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
