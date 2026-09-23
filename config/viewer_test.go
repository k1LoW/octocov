package config

import (
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v67/github"
	"github.com/k1LoW/go-github-client/v67/factory"
	"github.com/k1LoW/octocov/gh"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

func TestUnmarshalViewer(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    ViewerType
		wantErr string
	}{
		{"the type alone is shorthand for a map", "viewer: octocov.dev", ViewerOctocovDev, ""},
		{"artifact", "viewer: artifact", ViewerArtifact, ""},
		{"none", "viewer: none", ViewerNone, ""},
		{"a map holding the type", "viewer:\n  type: artifact", ViewerArtifact, ""},
		{"a custom viewer with its links", "viewer:\n  type: custom\n  links:\n    coverage: '\"https://example.com\"'", ViewerCustom, ""},
		{"a custom viewer linking nothing", "viewer:\n  type: custom", ViewerCustom, ""},
		{"an unknown type", "viewer: codecov", "", `invalid viewer.type: "codecov"`},
		{"no type", "viewer:\n  links:\n    coverage: '\"x\"'", "", "viewer.type: is not set"},
		{"links are for a custom viewer only", "viewer:\n  type: artifact\n  links:\n    coverage: '\"x\"'", "", "viewer.links: is only for viewer.type: custom"},
		{"a link that does not compile", "viewer:\n  type: custom\n  links:\n    coverageFile: 'file.path +'", "", "viewer.links.coverageFile:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			err := yaml.Unmarshal([]byte(tt.in), c)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got %v\nwant an error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.Viewer == nil || c.Viewer.Type != tt.want {
				t.Errorf("got %v\nwant %v", c.Viewer, tt.want)
			}
		})
	}
}

func TestCustomLinksLink(t *testing.T) {
	c := New()
	in := `viewer:
  type: custom
  links:
    coverage: 'report.is_base ? nil : "https://example.com/" + env.REPO + "/" + report.ref'
    coverageFile: 'file.path startsWith "gen/" ? "" : "https://example.com/" + pathEscape(file.path) + "?c=" + queryEscape(report.commit)'
    codeToTestRatio: '1'
`
	if err := yaml.Unmarshal([]byte(in), c); err != nil {
		t.Fatal(err)
	}
	l := &CustomLinks{programs: c.Viewer.programs, vars: map[string]any{"env": map[string]string{"REPO": "k1LoW/octocov"}}}
	report := func(isBase bool) map[string]any {
		return map[string]any{"ref": "refs/pull/1/merge", "commit": "a b", "is_base": isBase}
	}
	tests := []struct {
		name    string
		key     string
		vars    map[string]any
		want    string
		wantErr bool
	}{
		{"a string is the link", LinkCoverage, map[string]any{"report": report(false)}, "https://example.com/k1LoW/octocov/refs/pull/1/merge", false},
		{"nil is no link", LinkCoverage, map[string]any{"report": report(true)}, "", false},
		{"the escape functions", LinkCoverageFile, map[string]any{"report": report(false), "file": map[string]any{"path": "a b/c.go"}}, "https://example.com/a%20b%2Fc.go?c=a+b", false},
		{"an empty string is no link", LinkCoverageFile, map[string]any{"report": report(false), "file": map[string]any{"path": "gen/a.go"}}, "", false},
		{"a key left out links nothing", LinkTestExecutionTime, map[string]any{"report": report(false)}, "", false},
		{"anything else is an error", LinkCodeToTestRatio, map[string]any{"report": report(false)}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := l.Link(tt.key, tt.vars)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v\nwant one: %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestResolveViewer(t *testing.T) {
	tests := []struct {
		name       string
		viewer     *Viewer
		serverURL  string
		datastores []string
		private    bool
		want       ViewerType
	}{
		{"what is set wins", &Viewer{Type: ViewerNone}, "", []string{"artifact://k1LoW/octocov"}, false, ViewerNone},
		{"a public repository storing its report in an artifact", nil, "", []string{"artifact://k1LoW/octocov"}, false, ViewerOctocovDev},
		{"a private repository", nil, "", []string{"artifact://k1LoW/octocov"}, true, ViewerArtifact},
		{"a report stored in no artifact", nil, "", []string{"local://reports"}, false, ViewerArtifact},
		{"GitHub Enterprise Server", nil, "https://github.example.com", []string{"artifact://k1LoW/octocov"}, false, ViewerArtifact},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			c := &Config{
				Repository: "k1LoW/octocov",
				Report:     &Report{Datastores: tt.datastores},
				Viewer:     tt.viewer,
				gh:         mockedRepositoryGh(t, tt.private),
			}
			if got := c.ResolveViewer(t.Context()); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func mockedRepositoryGh(t *testing.T, private bool) *gh.Gh {
	t.Helper()
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposByOwnerByRepo,
			github.Repository{Private: new(private)},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := gh.New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)
	return g
}
