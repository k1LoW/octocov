package central

import (
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

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
)

//go:embed index.md.tmpl
var indexTmpl []byte

type Central struct {
	config  *CentralConfig
	reports []*report.Report
}

type CentralConfig struct {
	Repository             string
	Wd                     string
	Index                  string
	Badges                 string
	Reports                []fs.FS
	CoverageColor          func(cover float64) string
	CodeToTestRatioColor   func(ratio float64) string
	TestExecutionTimeColor func(d time.Duration) string
}

func New(c *CentralConfig) *Central {
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

func (c *Central) collectReports() error {
	rsMap := map[string]*report.Report{}

	// collect reports
	for _, fsys := range c.config.Reports {
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
	}

	for _, r := range rsMap {
		c.reports = append(c.reports, r)
	}
	sort.Slice(c.reports, func(i, j int) bool { return c.reports[i].Repository < c.reports[j].Repository })
	return nil
}

func (c *Central) generateBadges() ([]string, error) {
	generatedPaths := []string{}

	for _, r := range c.reports {
		cp := r.CoveragePercent()
		err := os.MkdirAll(filepath.Join(c.config.Badges, r.Repository), 0755) // #nosec
		if err != nil {
			return nil, err
		}
		bp := filepath.Join(c.config.Badges, r.Repository, "coverage.svg")
		out, err := os.OpenFile(bp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
		if err != nil {
			return nil, err
		}
		b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
		b.MessageColor = c.config.CoverageColor(cp)
		if err := b.Render(out); err != nil {
			return nil, err
		}
		generatedPaths = append(generatedPaths, bp)

		// Code to Test Ratio
		if r.CodeToTestRatio != nil {
			tr := r.CodeToTestRatioRatio()
			err := os.MkdirAll(filepath.Join(c.config.Badges, r.Repository), 0755) // #nosec
			if err != nil {
				return nil, err
			}
			bp := filepath.Join(c.config.Badges, r.Repository, "ratio.svg")
			out, err = os.OpenFile(bp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
			if err != nil {
				return nil, err
			}
			b := badge.New("code to test ratio", fmt.Sprintf("1:%.1f", tr))
			b.MessageColor = c.config.CodeToTestRatioColor(tr)
			if err := b.Render(out); err != nil {
				return nil, err
			}
			generatedPaths = append(generatedPaths, bp)
		}

		// Test Execution Time
		if r.TestExecutionTime != nil {
			d := time.Duration(*r.TestExecutionTime)
			err := os.MkdirAll(filepath.Join(c.config.Badges, r.Repository), 0755) // #nosec
			if err != nil {
				return nil, err
			}
			bp := filepath.Join(c.config.Badges, r.Repository, "time.svg")
			out, err = os.OpenFile(bp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
			if err != nil {
				return nil, err
			}
			b := badge.New("test execution time", d.String())
			b.MessageColor = c.config.TestExecutionTimeColor(d)
			if err := b.Render(out); err != nil {
				return nil, err
			}
			generatedPaths = append(generatedPaths, bp)
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
	owner, repo, err := gh.SplitRepository(c.config.Repository)
	if err != nil {
		return err
	}
	rawRootURL, err := g.GetRawRootURL(ctx, owner, repo)
	if err != nil {
		return err
	}

	// Get project root dir
	proot := c.config.Wd

	croot := c.config.Index
	if strings.HasSuffix(croot, ".md") {
		croot = filepath.Dir(c.config.Index)
	}

	badgesLinkRel, err := filepath.Rel(croot, c.config.Badges)
	if err != nil {
		return err
	}

	badgesURLRel, err := filepath.Rel(proot, c.config.Badges)
	if err != nil {
		return err
	}

	d := map[string]interface{}{
		"Host":          host,
		"Reports":       c.reports,
		"BadgesLinkRel": badgesLinkRel,
		"BadgesURLRel":  badgesURLRel,
		"RawRootURL":    rawRootURL,
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
			return time.Duration(*r.TestExecutionTime).String()
		},
	}
}
