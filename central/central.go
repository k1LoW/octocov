package central

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/local"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

//go:embed index.md.tmpl
var indexTmpl []byte

type Central struct {
	config  *Config
	reports []*report.Report
	// artifactBacked says, per repository, whether the collected report was read out of a
	// GitHub Actions artifact. The pages that browse a report read it out of the artifacts of
	// the repository it describes, so collecting from one is the index's only proof that such
	// a page exists. A report read from anywhere else may still have one, and may not, and
	// nothing here can tell the two apart.
	artifactBacked map[string]bool
	// stderr is where the warnings about what could not be collected go. It is a field so a
	// test can read them back, since a warning nobody can see is the state this replaced.
	stderr io.Writer
}

type Config struct {
	Repository             string
	Wd                     string
	Index                  string
	Badges                 []datastore.Datastore
	Reports                []ReportDatastore
	CoverageBadge          BadgeRenderer
	CodeToTestRatioBadge   BadgeRenderer
	TestExecutionTimeBadge BadgeRenderer
	// BadgeViewer is where the badges of the index link to, and nil links none.
	BadgeViewer *report.Viewer
}

// BadgeRenderer renders the badge of a measured value to w. What the badge says about the
// value, and how it is decorated, belongs to the configuration of the repository running
// central mode, which is read outside of this package.
type BadgeRenderer func(w io.Writer, value float64) error

// ReportDatastore is a datastore the index is collected from, named by the URL it was
// configured with. A Datastore cannot say which line of the config produced it, and a
// warning about one that could not be read is of no use without that name.
type ReportDatastore struct {
	URL       string
	Datastore datastore.Datastore
}

func New(c *Config) *Central {
	return &Central{
		config: c,
		stderr: os.Stderr,
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
	backed := map[string]bool{}

	// collect reports
	failed := 0
	for _, rd := range c.config.Reports {
		fromArtifact := isArtifact(rd.Datastore)
		fsys, err := rd.Datastore.FS()
		if err != nil {
			c.warnSkippedDatastore(rd.URL, err)
			failed++
			continue
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
			// The datastore also holds the reports of the pull requests and the branches
			// of each repository. The index describes the state of the default branch, so
			// only the reports stored under its key belong in it.
			if r.RefKey() != "" {
				return nil
			}
			current, ok := rsMap[r.Repository]
			if !ok {
				if _, err := fmt.Fprintf(c.stderr, "Collect report of %s\n", r.Repository); err != nil {
					return err
				}
				rsMap[r.Repository] = r
				backed[r.Repository] = fromArtifact
				return nil
			}
			if current.Timestamp.UnixNano() < r.Timestamp.UnixNano() {
				rsMap[r.Repository] = r
				backed[r.Repository] = fromArtifact
			}
			return nil
		}); err != nil {
			// Whatever the walk reached before it stopped is kept, since a half-collected
			// datastore still describes the repositories it did reach.
			c.warnSkippedDatastore(rd.URL, err)
			failed++
			continue
		}
	}

	// The index is rewritten from what was collected, so a failure that leaves nothing to
	// write with would replace the whole of it with an empty page, which is the state the
	// warnings were meant to make visible. What decides it is whether anything was
	// collected rather than how many datastores failed, since a walk that stops partway
	// still contributes the repositories it reached and those belong in the index.
	if failed > 0 && len(rsMap) == 0 {
		return fmt.Errorf("could not collect any report, and %d of the %d datastore(s) could not be read", failed, len(c.config.Reports))
	}

	for _, r := range rsMap {
		c.reports = append(c.reports, r)
	}
	c.artifactBacked = backed
	sort.Slice(c.reports, func(i, j int) bool { return c.reports[i].Repository < c.reports[j].Repository })
	return nil
}

func (c *Central) generateBadges() ([]string, error) {
	ctx := context.Background()
	badges := map[string][]byte{}
	for _, r := range c.reports {
		bp := filepath.Join(r.Repository, "coverage.svg")
		out := new(bytes.Buffer)
		if err := c.config.CoverageBadge(out, r.CoveragePercent()); err != nil {
			return nil, err
		}
		badges[bp] = out.Bytes()

		// Code to Test Ratio
		if r.CodeToTestRatio != nil {
			bp := filepath.Join(r.Repository, "ratio.svg")
			out := new(bytes.Buffer)
			if err := c.config.CodeToTestRatioBadge(out, r.CodeToTestRatioRatio()); err != nil {
				return nil, err
			}
			badges[bp] = out.Bytes()
		}

		// Test Execution Time
		if r.TestExecutionTime != nil {
			bp := filepath.Join(r.Repository, "time.svg")
			out := new(bytes.Buffer)
			if err := c.config.TestExecutionTimeBadge(out, r.TestExecutionTimeNano()); err != nil {
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
	tmpl := template.Must(template.New("index").Funcs(c.funcs()).Parse(string(indexTmpl)))
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
		rootURL = fmt.Sprintf("%s/%s/%s/blob/%s", host, repo.Owner, repo.Repo, b)
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
		"BadgesLinkRel": filepath.ToSlash(badgesLinkRel),
		"BadgesURLRel":  filepath.ToSlash(badgesURLRel),
		"RootURL":       rootURL,
		"IsPrivate":     isPrivate,
		"Query":         query,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}

	return nil
}

func (c *Central) funcs() map[string]any {
	return template.FuncMap{
		"coverage": func(r *report.Report) string {
			return fmt.Sprintf("%.1f%%", floor1(r.CoveragePercent()))
		},
		"ratio": func(r *report.Report) string {
			if r.CodeToTestRatio == nil {
				return "-"
			}
			return fmt.Sprintf("1:%.1f", floor1(r.CodeToTestRatioRatio()))
		},
		"time": func(r *report.Report) string {
			if r.TestExecutionTime == nil {
				return "-"
			}
			return time.Duration(r.TestExecutionTimeNano()).String()
		},
		"badge": func(r *report.Report, kind, alt, src string) (string, error) {
			img := fmt.Sprintf("![%s](%s)", alt, src)
			v := c.config.BadgeViewer
			var u string
			switch {
			case v.ReadsArtifacts():
				if !c.artifactBacked[r.Repository] {
					return img, nil
				}
				// The pages have one per report rather than one per metric, so every badge
				// opens the same one.
				u = v.CoverageURL(r)
			case kind == "coverage":
				u = v.CoverageURL(r)
			case kind == "ratio":
				u = v.CodeToTestRatioURL(r)
			case kind == "time":
				u = v.TestExecutionTimeURL(r)
			default:
				return "", fmt.Errorf("unknown badge: %s", kind)
			}

			if err := v.Err(); err != nil {
				return "", err
			}
			if u == "" {
				return img, nil
			}
			return fmt.Sprintf("[%s](%s)", img, u), nil
		},
	}
}

// artifactDatastore is the datastore that reads its reports out of GitHub Actions artifacts.
type artifactDatastore interface {
	datastore.Datastore
	IsArtifact() bool
}

// isArtifact reports whether d reads its reports out of GitHub Actions artifacts, which is
// the same place the pages that browse a report read it from.
func isArtifact(d datastore.Datastore) bool {
	a, ok := d.(artifactDatastore)
	return ok && a.IsArtifact()
}

// warnSkippedDatastore reports a datastore the index could not be collected from. One
// unreadable datastore is not a reason to drop the repositories the others describe, so it
// is a warning rather than a failure.
func (c *Central) warnSkippedDatastore(u string, err error) {
	if u == "" {
		u = "datastore"
	}
	fmt.Fprintf(c.stderr, "Skip collecting reports from %s: %v\n", u, err) //nostyle:handlerrors
}

// floor1 round down to one decimal place.
func floor1(v float64) float64 {
	return math.Floor(v*10) / 10
}
