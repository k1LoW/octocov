package central

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/local"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
)

//go:embed index.md.tmpl
var indexTmpl []byte

type Central struct {
	config  *Config
	reports []*report.Report
}

type Config struct {
	Repository             string
	Wd                     string
	Index                  string
	Badges                 []datastore.Datastore
	Reports                []datastore.Datastore
	CoverageColor          func(cover float64) string
	CodeToTestRatioColor   func(ratio float64) string
	TestExecutionTimeColor func(d time.Duration) string
}

func New(c *Config) *Central {
	return &Central{
		config: c,
	}
}

func (c *Central) Generate(ctx context.Context) ([]string, error) {
	// collect reports
	if err := c.collectReports(); err != nil {
		return nil, err
	}

	// generate badges
	paths, err := c.generateBadges()
	if err != nil {
		return nil, err
	}

	// render index
	p := c.config.Index
	fi, err := os.Stat(c.config.Index)
	if err == nil && fi.IsDir() {
		p = filepath.Join(c.config.Index, "README.md")
	}
	i, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
	if err != nil {
		return nil, err
	}
	if err := c.renderIndex(i); err != nil {
		return nil, err
	}
	paths = append(paths, p)

	return paths, nil
}

func (c *Central) CollectedReports() []*report.Report {
	return c.reports
}

func (c *Central) collectReports() error {
	rsMap := map[string]*report.Report{}

	// collect reports
	for _, d := range c.config.Reports {
		fsys, err := d.FS()
		if err != nil {
			return err
		}
		if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
				return nil
			}
			r := &report.Report{}
			f, err := fsys.Open(path)
			if err != nil {
				return nil
			}
			defer f.Close()
			b, err := io.ReadAll(f)
			if err != nil {
				return nil
			}
			if err := json.Unmarshal(b, r); err != nil {
				return nil
			}
			current, ok := rsMap[r.Repository]
			if !ok {
				if _, err := fmt.Fprintf(os.Stderr, "Collect report of %s\n", r.Repository); err != nil {
					return err
				}
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
	}

	for _, r := range rsMap {
		c.reports = append(c.reports, r)
	}
	sort.Slice(c.reports, func(i, j int) bool { return c.reports[i].Repository < c.reports[j].Repository })
	return nil
}

func (c *Central) generateBadges() ([]string, error) {
	ctx := context.Background()
	badges := map[string][]byte{}
	for _, r := range c.reports {
		cp := r.CoveragePercent()
		bp := filepath.Join(r.Repository, "coverage.svg")
		out := new(bytes.Buffer)
		b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
		b.MessageColor = c.config.CoverageColor(cp)
		if err := b.AddIcon(internal.Icon); err != nil {
			return nil, err
		}
		if err := b.Render(out); err != nil {
			return nil, err
		}
		badges[bp] = out.Bytes()

		// Code to Test Ratio
		if r.CodeToTestRatio != nil {
			tr := r.CodeToTestRatioRatio()
			bp := filepath.Join(r.Repository, "ratio.svg")
			out := new(bytes.Buffer)
			b := badge.New("code to test ratio", fmt.Sprintf("1:%.1f", tr))
			b.MessageColor = c.config.CodeToTestRatioColor(tr)
			if err := b.AddIcon(internal.Icon); err != nil {
				return nil, err
			}
			if err := b.Render(out); err != nil {
				return nil, err
			}
			badges[bp] = out.Bytes()
		}

		// Test Execution Time
		if r.TestExecutionTime != nil {
			d := time.Duration(r.TestExecutionTimeNano())
			bp := filepath.Join(r.Repository, "time.svg")
			out := new(bytes.Buffer)
			b := badge.New("test execution time", d.String())
			b.MessageColor = c.config.TestExecutionTimeColor(d)
			if err := b.AddIcon(internal.Icon); err != nil {
				return nil, err
			}
			if err := b.Render(out); err != nil {
				return nil, err
			}
			badges[bp] = out.Bytes()
		}
	}
	var generatedPaths []string
	for _, d := range c.config.Badges {
		for path, content := range badges {
			if err := d.Put(ctx, path, content); err != nil {
				return nil, err
			}
			switch v := d.(type) {
			case *local.Local:
				generatedPaths = append(generatedPaths, filepath.Join(v.Root(), path))
			}
		}
	}
	return generatedPaths, nil
}

func (c *Central) renderIndex(wr io.Writer) error {
	tmpl := template.Must(template.New("index").Funcs(funcs()).Parse(string(indexTmpl)))
	host := os.Getenv("GITHUB_SERVER_URL")
	if host == "" {
		host = gh.DefaultGithubServerURL
	}

	ctx := context.Background()
	g, err := gh.New()
	if err != nil {
		return err
	}
	repo, err := gh.Parse(c.config.Repository)
	if err != nil {
		return err
	}
	isPrivate, err := g.IsPrivate(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return err
	}
	var (
		rootURL string
		query   string
	)
	if !isPrivate {
		rootURL, err = g.FetchRawRootURL(ctx, repo.Owner, repo.Repo)
		if err != nil {
			return err
		}
	} else {
		b, err := g.FetchDefaultBranch(ctx, repo.Owner, repo.Repo)
		if err != nil {
			return err
		}
		rootURL = fmt.Sprintf("%s/%s/%s/blob/%s/", host, repo.Owner, repo.Repo, b)
		query = "?raw=true"
	}

	// Get project root dir
	proot := c.config.Wd

	croot := c.config.Index
	if strings.HasSuffix(croot, ".md") {
		croot = filepath.Dir(c.config.Index)
	}

	var broot string
	for _, d := range c.config.Badges {
		switch v := d.(type) {
		case *local.Local:
			broot = v.Root()
		}
	}

	badgesLinkRel, err := filepath.Rel(croot, broot)
	if err != nil {
		return err
	}

	badgesURLRel, err := filepath.Rel(proot, broot)
	if err != nil {
		return err
	}

	d := map[string]any{
		"Host":          host,
		"Reports":       c.reports,
		"BadgesLinkRel": badgesLinkRel,
		"BadgesURLRel":  badgesURLRel,
		"RootURL":       rootURL,
		"IsPrivate":     isPrivate,
		"Query":         query,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}

	return nil
}

func funcs() map[string]any {
	return template.FuncMap{
		"coverage": func(r *report.Report) string {
			return fmt.Sprintf("%.1f%%", r.CoveragePercent())
		},
		"ratio": func(r *report.Report) string {
			if r.CodeToTestRatio == nil {
				return "-"
			}
			return fmt.Sprintf("1:%.1f", r.CodeToTestRatioRatio())
		},
		"time": func(r *report.Report) string {
			if r.TestExecutionTime == nil {
				return "-"
			}
			return time.Duration(r.TestExecutionTimeNano()).String()
		},
	}
}
