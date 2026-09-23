package config

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"os"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/goccy/go-yaml"
	"github.com/k1LoW/octocov/gh"
)

type ViewerType string

const (
	ViewerOctocovDev ViewerType = "octocov.dev"
	ViewerArtifact   ViewerType = "artifact"
	ViewerNone       ViewerType = "none"
	ViewerCustom     ViewerType = "custom"
)

// The keys of `viewer.links:`, one for each kind of value a table can link.
const (
	LinkCoverage          = "coverage"
	LinkCoverageFile      = "coverageFile"
	LinkCodeToTestRatio   = "codeToTestRatio"
	LinkTestExecutionTime = "testExecutionTime"
)

// Viewer says where the values in the tables link to.
type Viewer struct {
	Type  ViewerType   `yaml:"type"`
	Links *ViewerLinks `yaml:"links,omitempty"`

	programs map[string]*vm.Program
}

type ViewerLinks struct {
	Coverage          string `yaml:"coverage,omitempty"`
	CoverageFile      string `yaml:"coverageFile,omitempty"`
	CodeToTestRatio   string `yaml:"codeToTestRatio,omitempty"`
	TestExecutionTime string `yaml:"testExecutionTime,omitempty"`
}

// UnmarshalYAML reads a viewer written as its type alone, which is shorthand for a map
// holding only `type`.
func (v *Viewer) UnmarshalYAML(data []byte) error {
	var t string
	if err := yaml.Unmarshal(data, &t); err == nil {
		v.Type = ViewerType(t)
		return v.validate()
	}
	s := struct {
		Type  ViewerType   `yaml:"type"`
		Links *ViewerLinks `yaml:"links,omitempty"`
	}{}
	if err := yaml.Unmarshal(data, &s); err != nil {
		return err
	}
	v.Type = s.Type
	v.Links = s.Links
	return v.validate()
}

// ResolveViewer returns the viewer the tables link to. When `viewer:` is not set, a public
// repository on github.com whose report is stored in an artifact links to octocov.dev,
// since there it asks nothing of a reader, and everything else links to the page rendered
// into an artifact.
func (c *Config) ResolveViewer(ctx context.Context) ViewerType {
	if c.Viewer != nil {
		return c.Viewer.Type
	}
	if c.canUseOctocovDev(ctx) {
		return ViewerOctocovDev
	}
	return ViewerArtifact
}

func (c *Config) canUseOctocovDev(ctx context.Context) bool {
	if s := strings.TrimRight(os.Getenv("GITHUB_SERVER_URL"), "/"); s != "" && s != gh.DefaultGithubServerURL {
		return false
	}
	if c.Report == nil || !hasArtifactDatastore(c.Report.Datastores) {
		return false
	}
	return c.isPublic(ctx)
}

// ResolveCentralViewer returns the viewer the badges of the central mode link to. The
// reports it collects are named by the central repository, so its visibility stands in for
// theirs when `viewer:` is not set.
func (c *Config) ResolveCentralViewer(ctx context.Context) ViewerType {
	if c.Viewer != nil {
		return c.Viewer.Type
	}
	if s := strings.TrimRight(os.Getenv("GITHUB_SERVER_URL"), "/"); s != "" && s != gh.DefaultGithubServerURL {
		return ViewerArtifact
	}
	if c.isPublic(ctx) {
		return ViewerOctocovDev
	}
	return ViewerArtifact
}

// isPublic reports whether the repository is public, and answers false when that cannot
// be told, since a private repository linked to octocov.dev asks a reader for more than one
// left unlinked does.
func (c *Config) isPublic(ctx context.Context) bool {
	repo, err := gh.Parse(c.Repository)
	if err != nil {
		return false
	}
	if c.gh == nil {
		g, err := gh.New()
		if err != nil {
			return false
		}
		c.gh = g
	}
	private, err := c.gh.IsPrivate(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return false
	}
	return !private
}

func hasArtifactDatastore(datastores []string) bool {
	for _, d := range datastores {
		if strings.HasPrefix(d, "artifact://") {
			return true
		}
	}
	return false
}

// CustomLinks evaluates the expressions of `viewer.links:`.
type CustomLinks struct {
	programs map[string]*vm.Program
	vars     map[string]any
}

// CustomLinks returns the expressions of `viewer.links:` bound to the variables `if:` has,
// or nil when the viewer is not a custom one.
func (c *Config) CustomLinks() (*CustomLinks, error) {
	if c.Viewer == nil || c.Viewer.Type != ViewerCustom {
		return nil, nil
	}
	vars, err := c.ifVariables()
	if err != nil {
		return nil, err
	}
	return c.Viewer.CustomLinks(vars), nil
}

// CustomLinks returns the expressions of `viewer.links:` bound to vars.
func (v *Viewer) CustomLinks(vars map[string]any) *CustomLinks {
	return &CustomLinks{programs: v.programs, vars: vars}
}

// Link evaluates the expression of key with vars joined to the variables `if:` has, and
// returns the link it answers, or an empty string when there is none.
func (l *CustomLinks) Link(key string, vars map[string]any) (string, error) {
	if l == nil {
		return "", nil
	}
	p, ok := l.programs[key]
	if !ok {
		return "", nil
	}
	env := make(map[string]any, len(l.vars)+len(vars))
	maps.Copy(env, l.vars)
	maps.Copy(env, vars)
	out, err := expr.Run(p, env)
	if err != nil {
		return "", fmt.Errorf("viewer.links.%s: %w", key, err)
	}
	switch v := out.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("viewer.links.%s: the link is %T, not a string", key, out)
	}
}

func (v *Viewer) validate() error {
	switch v.Type {
	case ViewerOctocovDev, ViewerArtifact, ViewerNone:
		if v.Links != nil {
			return fmt.Errorf("viewer.links: is only for viewer.type: %s", ViewerCustom)
		}
		return nil
	case ViewerCustom:
		return v.compile()
	case "":
		return errors.New("viewer.type: is not set")
	default:
		return fmt.Errorf("invalid viewer.type: %q", v.Type)
	}
}

var linkFunctions = []expr.Option{
	expr.Function("pathEscape", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("pathEscape takes a string, not %T", params[0])
		}
		return url.PathEscape(s), nil
	}, new(func(string) string)),
	expr.Function("queryEscape", func(params ...any) (any, error) {
		s, ok := params[0].(string)
		if !ok {
			return nil, fmt.Errorf("queryEscape takes a string, not %T", params[0])
		}
		return url.QueryEscape(s), nil
	}, new(func(string) string)),
}

func (v *Viewer) compile() error {
	v.programs = map[string]*vm.Program{}
	if v.Links == nil {
		return nil
	}
	for key, src := range map[string]string{
		LinkCoverage:          v.Links.Coverage,
		LinkCoverageFile:      v.Links.CoverageFile,
		LinkCodeToTestRatio:   v.Links.CodeToTestRatio,
		LinkTestExecutionTime: v.Links.TestExecutionTime,
	} {
		if src == "" {
			continue
		}
		p, err := expr.Compile(src, linkFunctions...)
		if err != nil {
			return fmt.Errorf("viewer.links.%s: %w", key, err)
		}
		v.programs[key] = p
	}
	return nil
}
