package gh

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v67/github"
	"github.com/k1LoW/go-github-client/v67/factory"
	"github.com/migueleliasweb/go-github-mock/src/mock"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    *Repository
		wantErr bool
	}{
		{"owner/repo", &Repository{Owner: "owner", Repo: "repo"}, false},
		{"owner/repo/path/to", &Repository{Owner: "owner", Repo: "repo", Path: "path/to"}, false},
		{"owner/repo@sub", &Repository{Owner: "owner", Repo: "repo@sub"}, false},
		{"owner/repo.sub", &Repository{Owner: "owner", Repo: "repo.sub"}, false},
		{"owner/../sub", nil, true},
		{"owner", nil, true},
		{"owner/../sub", nil, true},
		{"owner/./sub", nil, true},
		{"owner//sub", nil, true},
		{"owner/repo/sub/", nil, true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got error %v\n", err)
			}
			continue
		}
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Error(diff)
		}
	}
}

func TestFetchDefaultBranch(t *testing.T) {
	mg := mockedGh(t)
	want := "main"
	got, err := mg.FetchDefaultBranch(t.Context(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestFetchRawRootURL(t *testing.T) {
	// The case under test is the github.com one, which an ambient value naming a GitHub
	// Enterprise Server would answer before the tree is ever read.
	t.Setenv("GITHUB_SERVER_URL", DefaultGithubServerURL)
	ctx := t.Context()
	token, _, _, _ := factory.GetTokenAndEndpoints()
	if token == "" {
		t.Skip("no token")
		return
	}
	tests := []struct {
		owner string
		repo  string
		want  string
	}{
		{"k1LoW", "octocov", "https://raw.githubusercontent.com/k1LoW/octocov/main"},
	}
	for _, tt := range tests {
		g, err := New()
		if err != nil {
			t.Fatal(err)
		}
		got, err := g.FetchRawRootURL(ctx, tt.owner, tt.repo)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestDetectCurrentBranch(t *testing.T) {
	tests := []struct {
		GITHUB_REF      string
		GITHUB_HEAD_REF string
		want            string
		wantErr         bool
	}{
		{"refs/pull/8/head", "", "", true},
		// The shape of a pull_request_target run, where GITHUB_REF names the base branch
		// and only GITHUB_HEAD_REF names the branch being built.
		{"refs/heads/name", "mybranch", "mybranch", false},
		{"refs/heads/branch/branch/name", "", "branch/branch/name", false},
		{"refs/pull/8/head", "mybranch", "mybranch", false},
	}
	ctx := t.Context()
	mg := mockedGh(t)
	for _, tt := range tests {
		t.Run(tt.GITHUB_REF, func(t *testing.T) {
			t.Setenv("GITHUB_REF", tt.GITHUB_REF)
			t.Setenv("GITHUB_HEAD_REF", tt.GITHUB_HEAD_REF)
			got, err := mg.DetectCurrentBranch(ctx)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got err: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want err")
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestDetectCurrentPullRequestNumber(t *testing.T) {
	tests := []struct {
		name                       string
		GITHUB_PULL_REQUEST_NUMBER string
		GITHUB_REF                 string
		GITHUB_HEAD_REF            string
		want                       int
		wantErr                    bool
	}{
		{"pull request ref", "", "refs/pull/8/head", "", 8, false},
		{"branch ref", "", "refs/heads/branch/branch/name", "", 13, false},
		{"malformed ref", "", "refs/8", "", 0, true},
		{"number from the environment", "8", "", "", 8, false},
		{"unparsable number", "str", "", "", 0, true},
		// On pull_request_target GITHUB_REF names the base branch, so the search has to
		// go after the head ref rather than after whatever GITHUB_REF holds.
		{"base branch ref with a head ref", "", "refs/heads/main", "branch/branch/name", 13, false},
	}
	ctx := t.Context()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Per case, since the mock serves each matched request once and more than one
			// case now reaches the pull request listing.
			mg := mockedGh(t)
			t.Setenv("GITHUB_PULL_REQUEST_NUMBER", tt.GITHUB_PULL_REQUEST_NUMBER)
			t.Setenv("GITHUB_REF", tt.GITHUB_REF)
			t.Setenv("GITHUB_HEAD_REF", tt.GITHUB_HEAD_REF)
			got, err := mg.DetectCurrentPullRequestNumber(ctx, "owner", "repo")
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got err: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want err")
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestGenerateSig(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"", "<!-- octocov -->"},
		{"foo", "<!-- octocov:foo -->"},
	}
	for _, tt := range tests {
		got := generateSig(tt.key)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestInsertToBody(t *testing.T) {
	const content = "## Code Metrics Report\n| | main | PR |\n|-|-|-|"
	tests := []struct {
		name    string
		current string
		sig     string
		want    string
	}{
		{
			"empty body",
			"",
			"<!-- octocov -->",
			"<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"body without trailing newline",
			"Some PR description.",
			"<!-- octocov -->",
			"Some PR description.\n\n<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"body ending with an unterminated HTML block",
			"Some PR description.\n\n<a href=\"https://example.com\">\n  <img src=\"https://example.com/badge.svg\" alt=\"badge\">\n</a>\n",
			"<!-- octocov -->",
			"Some PR description.\n\n<a href=\"https://example.com\">\n  <img src=\"https://example.com/badge.svg\" alt=\"badge\">\n</a>\n\n<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"body already ending with a blank line",
			"Some PR description.\n\n",
			"<!-- octocov -->",
			"Some PR description.\n\n<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"body with CRLF line endings",
			"Some PR description.\r\n",
			"<!-- octocov -->",
			"Some PR description.\n\n<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"body with keyed sig",
			"Some PR description.",
			"<!-- octocov:foo -->",
			"Some PR description.\n\n<!-- octocov:foo -->\n" + content + "\n<!-- octocov:foo -->\n",
		},
		{
			"re-embed replaces the previously embedded content",
			"Some PR description.\n\n<!-- octocov -->\n## Old Report\n<!-- octocov -->\n",
			"<!-- octocov -->",
			"Some PR description.\n\n<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"re-embed restores the blank line missing from an already embedded body",
			"Some PR description.\n\n<a href=\"https://example.com\">\n</a>\n<!-- octocov -->\n## Old Report\n<!-- octocov -->\n",
			"<!-- octocov -->",
			"Some PR description.\n\n<a href=\"https://example.com\">\n</a>\n\n<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
		{
			"body consisting only of the embedded content",
			"<!-- octocov -->\n## Old Report\n<!-- octocov -->\n",
			"<!-- octocov -->",
			"<!-- octocov -->\n" + content + "\n<!-- octocov -->\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := insertToBody(tt.current, content, tt.sig)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}
func mockedGh(t *testing.T) *Gh {
	t.Setenv("GITHUB_TOKEN", "dummy")
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposByOwnerByRepo,
			github.Repository{
				DefaultBranch: new("main"),
			},
		),
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposPullsByOwnerByRepo,
			[]*github.PullRequest{
				{
					Head: &github.PullRequestBranch{
						Ref: new("branch/branch/name"),
						Repo: &github.Repository{
							Owner: &github.User{Login: new("owner")},
							Name:  new("repo"),
						},
					},
					Number: new(13),
				},
			},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)
	return g
}

func TestDetectCurrentPullRequestNumberSkipsForkPR(t *testing.T) {
	tests := []struct {
		name       string
		GITHUB_REF string
		prs        []*github.PullRequest
		want       int
		wantErr    bool
	}{
		{
			name:       "same repo PR is detected",
			GITHUB_REF: "refs/heads/feature-branch",
			prs: []*github.PullRequest{
				{
					Number: new(10),
					Head: &github.PullRequestBranch{
						Ref: new("feature-branch"),
						Repo: &github.Repository{
							Owner: &github.User{Login: new("owner")},
							Name:  new("repo"),
						},
					},
				},
			},
			want:    10,
			wantErr: false,
		},
		{
			name:       "fork PR is skipped",
			GITHUB_REF: "refs/heads/main",
			prs: []*github.PullRequest{
				{
					Number: new(20),
					Head: &github.PullRequestBranch{
						Ref: new("main"),
						Repo: &github.Repository{
							Owner: &github.User{Login: new("forked-user")},
							Name:  new("repo"),
						},
					},
				},
			},
			want:    0,
			wantErr: true,
		},
		{
			name:       "fork PR is skipped, same repo PR is detected",
			GITHUB_REF: "refs/heads/main",
			prs: []*github.PullRequest{
				{
					Number: new(20),
					Head: &github.PullRequestBranch{
						Ref: new("main"),
						Repo: &github.Repository{
							Owner: &github.User{Login: new("forked-user")},
							Name:  new("repo"),
						},
					},
				},
				{
					Number: new(30),
					Head: &github.PullRequestBranch{
						Ref: new("main"),
						Repo: &github.Repository{
							Owner: &github.User{Login: new("owner")},
							Name:  new("repo"),
						},
					},
				},
			},
			want:    30,
			wantErr: false,
		},
		{
			name:       "owner case insensitive match",
			GITHUB_REF: "refs/heads/feature",
			prs: []*github.PullRequest{
				{
					Number: new(25),
					Head: &github.PullRequestBranch{
						Ref: new("feature"),
						Repo: &github.Repository{
							Owner: &github.User{Login: new("Owner")},
							Name:  new("repo"),
						},
					},
				},
			},
			want:    25,
			wantErr: false,
		},
		{
			name:       "nil head repo is skipped",
			GITHUB_REF: "refs/heads/feature",
			prs: []*github.PullRequest{
				{
					Number: new(35),
					Head: &github.PullRequestBranch{
						Ref:  new("feature"),
						Repo: nil,
					},
				},
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_PULL_REQUEST_NUMBER", "")
			t.Setenv("GITHUB_REF", tt.GITHUB_REF)
			t.Setenv("GITHUB_HEAD_REF", "")
			t.Setenv("GITHUB_TOKEN", "dummy")

			mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
				mock.WithRequestMatch( //nostyle:funcfmt
					mock.GetReposPullsByOwnerByRepo,
					tt.prs,
				),
			)
			client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			g, err := New()
			if err != nil {
				t.Fatal(err)
			}
			g.SetClient(client)

			got, err := g.DetectCurrentPullRequestNumber(t.Context(), "owner", "repo")
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got err: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want err")
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestListWorkflowJobs(t *testing.T) {
	// Every job of the run must be returned even when the run has more jobs
	// than fit in a single page of the API response.
	t.Setenv("GITHUB_TOKEN", "dummy")

	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatchPages( //nostyle:funcfmt
			mock.GetReposActionsRunsJobsByOwnerByRepoByRunId,
			github.Jobs{
				TotalCount: new(3),
				Jobs: []*github.WorkflowJob{
					{ID: github.Int64(1), Name: new("test (1)")},
					{ID: github.Int64(2), Name: new("test (2)")},
				},
			},
			github.Jobs{
				TotalCount: new(3),
				Jobs: []*github.WorkflowJob{
					{ID: github.Int64(3), Name: new("test (3)")},
				},
			},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	jobs, err := g.listWorkflowJobs(t.Context(), "owner", "repo", 1)
	if err != nil {
		t.Fatal(err)
	}
	var got []int64
	for _, j := range jobs {
		got = append(got, j.GetID())
	}
	want := []int64{1, 2, 3}
	if diff := cmp.Diff(got, want, nil); diff != "" {
		t.Error(diff)
	}
}

func TestFetchStepsByNameErrors(t *testing.T) {
	// Exhausting the retries must fail loudly. Reporting no steps as a success
	// would measure a test execution time of zero for a step that was merely
	// misspelled or still running.
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		jobs    []*github.WorkflowJob
		step    string
		wantErr string
	}{
		{
			name: "no job has a step with the given name",
			jobs: []*github.WorkflowJob{
				{ID: github.Int64(1), Steps: []*github.TaskStep{
					{
						Name:        new("Run test"),
						StartedAt:   &github.Timestamp{Time: base},
						CompletedAt: &github.Timestamp{Time: base.Add(time.Minute)},
					},
				}},
			},
			step:    "Run slow test",
			wantErr: `could not find any step named "Run slow test" in the workflow run`,
		},
		{
			name: "the named step never completes",
			jobs: []*github.WorkflowJob{
				{ID: github.Int64(1), Steps: []*github.TaskStep{
					{
						Name:      new("Run test"),
						StartedAt: &github.Timestamp{Time: base},
					},
				}},
			},
			step:    "Run test",
			wantErr: `step named "Run test" did not complete in time`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", "dummy")
			t.Setenv("GITHUB_RUN_ID", "1")

			mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
				mock.WithRequestMatch( //nostyle:funcfmt
					mock.GetReposActionsRunsJobsByOwnerByRepoByRunId,
					github.Jobs{TotalCount: new(len(tt.jobs)), Jobs: tt.jobs},
				),
			)
			client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			g, err := New()
			if err != nil {
				t.Fatal(err)
			}
			g.SetClient(client)

			// The retry window spans tens of seconds, so end it through the context
			// instead of waiting it out.
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()

			got, err := g.FetchStepsByName(ctx, "owner", "repo", tt.step)
			if err == nil {
				t.Fatalf("want err, got steps: %v", got)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("got %v\nwant %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsSameRepo(t *testing.T) {
	tests := []struct {
		name     string
		headRepo *github.Repository
		owner    string
		repo     string
		want     bool
	}{
		{
			name: "exact match",
			headRepo: &github.Repository{
				Owner: &github.User{Login: new("owner")},
				Name:  new("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  true,
		},
		{
			name: "owner case insensitive",
			headRepo: &github.Repository{
				Owner: &github.User{Login: new("Owner")},
				Name:  new("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  true,
		},
		{
			name: "different owner",
			headRepo: &github.Repository{
				Owner: &github.User{Login: new("forked-user")},
				Name:  new("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  false,
		},
		{
			name: "different repo name",
			headRepo: &github.Repository{
				Owner: &github.User{Login: new("owner")},
				Name:  new("other-repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  false,
		},
		{
			name:     "nil headRepo",
			headRepo: nil,
			owner:    "owner",
			repo:     "repo",
			want:     false,
		},
		{
			name: "nil owner in headRepo",
			headRepo: &github.Repository{
				Owner: nil,
				Name:  new("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSameRepo(tt.headRepo, tt.owner, tt.repo)
			if got != tt.want {
				t.Errorf("isSameRepo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseChangedLinesFromPatch(t *testing.T) {
	tests := []struct {
		name  string
		patch string
		want  []int
	}{
		{
			name:  "empty patch",
			patch: "",
			want:  nil,
		},
		{
			name:  "single addition",
			patch: "@@ -1,3 +1,4 @@\n unchanged\n-removed\n+added1\n+added2\n context",
			want:  []int{2, 3},
		},
		{
			name:  "pure deletion adds no lines",
			patch: "@@ -1,3 +1,1 @@\n unchanged\n-removed1\n-removed2",
			want:  nil,
		},
		{
			name:  "multiple hunks",
			patch: "@@ -1,2 +1,3 @@\n unchanged\n+added\n context\n@@ -10,2 +11,3 @@\n unchanged\n+added2\n context",
			want:  []int{2, 12},
		},
		{
			name:  "no newline at end of file marker is ignored",
			patch: "@@ -1,1 +1,2 @@\n unchanged\n+added\n\\ No newline at end of file",
			want:  []int{2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseChangedLinesFromPatch(tt.patch)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("got diff (-got +want):\n%s", diff)
			}
		})
	}
}

func TestFetchPullRequestFiles(t *testing.T) {
	// A file of six lines at the merge base. The pull request adds two lines after line 4, and
	// the base branch has since added three lines after line 1, so the lines the pull request
	// added are 5 and 6 in its head and 8 and 9 in the merge commit.
	const (
		headSHA    = "head"
		baseSHA    = "base"
		mergeSHA   = "merge"
		prPatch    = "@@ -4,3 +4,5 @@\n l4\n+a\n+b\n l5\n l6"
		mergePatch = "@@ -7,3 +7,5 @@\n l4\n+a\n+b\n l5\n l6"
	)
	tests := []struct {
		name        string
		commit      string
		parents     []string
		commitFails bool
		want        []int
		wantPatch   string
		wantParent  string
		wantAligned bool
	}{
		{
			name:        "the merge commit takes the lines it numbers",
			commit:      mergeSHA,
			parents:     []string{baseSHA, headSHA},
			want:        []int{8, 9},
			wantPatch:   mergePatch,
			wantParent:  baseSHA,
			wantAligned: true,
		},
		{
			name:        "the head keeps the lines of the pull request",
			commit:      headSHA,
			want:        []int{5, 6},
			wantPatch:   prPatch,
			wantAligned: true,
		},
		{
			name:      "a merge of another head keeps the lines of the pull request",
			commit:    mergeSHA,
			parents:   []string{baseSHA, "other"},
			want:      []int{5, 6},
			wantPatch: prPatch,
		},
		{
			name:        "a merge commit that cannot be looked up keeps the lines of the pull request",
			commit:      mergeSHA,
			commitFails: true,
			want:        []int{5, 6},
			wantPatch:   prPatch,
		},
		{
			name:        "no commit keeps the lines of the pull request",
			commit:      "",
			want:        []int{5, 6},
			wantPatch:   prPatch,
			wantAligned: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", "dummy")
			var parents []*github.Commit
			for _, p := range tt.parents {
				parents = append(parents, &github.Commit{SHA: new(p)})
			}
			mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
				mock.WithRequestMatch( //nostyle:funcfmt
					mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
					[]*github.CommitFile{{Filename: new("a.go"), Status: new("modified"), Patch: new(prPatch)}},
					[]*github.CommitFile{},
				),
				mock.WithRequestMatch( //nostyle:funcfmt
					mock.GetReposPullsByOwnerByRepoByPullNumber,
					github.PullRequest{Number: new(1), Head: &github.PullRequestBranch{SHA: new(headSHA)}},
				),
				mock.WithRequestMatchHandler( //nostyle:funcfmt
					mock.GetReposGitCommitsByOwnerByRepoByCommitSha,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if tt.commitFails {
							mock.WriteError(w, http.StatusInternalServerError, "boom")
							return
						}
						if _, err := w.Write(mock.MustMarshal(github.Commit{SHA: new(tt.commit), Parents: parents})); err != nil {
							t.Error(err)
						}
					}),
				),
				mock.WithRequestMatch( //nostyle:funcfmt
					mock.GetReposCompareByOwnerByRepoByBasehead,
					github.CommitsComparison{Files: []*github.CommitFile{{Filename: new("a.go"), Patch: new(mergePatch)}}},
				),
			)
			client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			g, err := New()
			if err != nil {
				t.Fatal(err)
			}
			g.SetClient(client)

			got, err := g.FetchPullRequestFiles(t.Context(), "owner", "repo", 1, tt.commit)
			if err != nil {
				t.Fatal(err)
			}
			files, unaligned := got.Files, got.Unaligned
			if len(files) != 1 {
				t.Fatalf("got %d files, want 1", len(files))
			}
			if diff := cmp.Diff(files[0].ChangedLines, tt.want); diff != "" {
				t.Errorf("got diff (-got +want):\n%s", diff)
			}
			// The page draws the patch beside the coverage, so it is numbered the same way.
			if files[0].Patch != tt.wantPatch {
				t.Errorf("got patch %q, want %q", files[0].Patch, tt.wantPatch)
			}
			if got.Parent != tt.wantParent {
				t.Errorf("got parent %q, want %q", got.Parent, tt.wantParent)
			}
			if got := unaligned == ""; got != tt.wantAligned {
				t.Errorf("got unaligned %q, want aligned %v", unaligned, tt.wantAligned)
			}
		})
	}
}

func TestAlignChangedLines(t *testing.T) {
	merged := func(n int) []*github.CommitFile {
		files := []*github.CommitFile{{
			Filename: new("a.go"), PreviousFilename: new("old.go"), Status: new("renamed"),
			Additions: new(1), Deletions: new(0), Patch: new("@@ -1,1 +1,2 @@\n l1\n+a"),
		}}
		for i := len(files); i < n; i++ {
			files = append(files, &github.CommitFile{Filename: new(strconv.Itoa(i) + ".go")})
		}
		return files
	}
	tests := []struct {
		name     string
		merged   []*github.CommitFile
		want     map[string][]int
		wantKept int
	}{
		{
			name:   "a file the merge leaves unchanged has no changed lines",
			merged: merged(1),
			want:   map[string][]int{"a.go": {2}, "b.go": nil},
		},
		{
			name:     "a file past the compare limit keeps the lines of the pull request",
			merged:   merged(compareFilesLimit),
			want:     map[string][]int{"a.go": {2}, "b.go": {7}},
			wantKept: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := []*PullRequestFile{
				{Filename: "a.go", Status: "modified", Additions: 3, Deletions: 2, ChangedLines: []int{5}},
				{Filename: "b.go", ChangedLines: []int{7}},
			}
			if kept := alignChangedLines(files, tt.merged); kept != tt.wantKept {
				t.Errorf("got %d files kept, want %d", kept, tt.wantKept)
			}
			got := map[string][]int{}
			for _, f := range files {
				got[f.Filename] = f.ChangedLines
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("got diff (-got +want):\n%s", diff)
			}
			// The file the merge diff names is described by that diff throughout, since the page
			// draws its counts and status beside the patch.
			if a := files[0]; a.PreviousFilename != "old.go" || a.Status != "renamed" || a.Additions != 1 || a.Deletions != 0 {
				t.Errorf("got %+v, want it described by the merge diff", a)
			}
			for _, f := range files {
				// Only a file the merge diff leaves out while it is under the limit is one the
				// merge changes nothing in.
				want := len(tt.want[f.Filename]) == 0
				if f.UnchangedByMerge != want {
					t.Errorf("%s: got UnchangedByMerge %v, want %v", f.Filename, f.UnchangedByMerge, want)
				}
			}
		})
	}
}

func TestChangedLinesByFile(t *testing.T) {
	files := []*PullRequestFile{
		{Filename: "a.go", ChangedLines: []int{1, 2}},
		{Filename: "b.go", ChangedLines: nil},
		{Filename: "c.go", ChangedLines: []int{5}},
	}
	got := ChangedLinesByFile(files)
	want := map[string][]int{
		"a.go": {1, 2},
		"c.go": {5},
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("got diff (-got +want):\n%s", diff)
	}
}

func TestChangedLinesByFileNoFiles(t *testing.T) {
	if got := ChangedLinesByFile(nil); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := ChangedLinesByFile([]*PullRequestFile{}); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestDetectCurrentPullRequestNumberClassifiesFailures(t *testing.T) {
	// Callers fall back to a default branch comparison when detection fails, and only report
	// the failures that are not simply "this run is not a pull request". The two must stay
	// distinguishable, because the fallback measures a different set of changed lines.
	tests := []struct {
		name               string
		GITHUB_REF         string
		wantNotPullRequest bool
	}{
		{"env is not set", "", true},
		{"pushed to a branch with no open pull request", "refs/heads/no-such-branch", true},
	}
	ctx := t.Context()
	mg := mockedGh(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_PULL_REQUEST_NUMBER", "")
			t.Setenv("GITHUB_REF", tt.GITHUB_REF)
			t.Setenv("GITHUB_HEAD_REF", "")
			_, err := mg.DetectCurrentPullRequestNumber(ctx, "owner", "repo")
			if err == nil {
				t.Fatal("want err")
			}
			if got := errors.Is(err, ErrNotPullRequest); got != tt.wantNotPullRequest {
				t.Errorf("errors.Is(err, ErrNotPullRequest) got %v, want %v (err: %v)", got, tt.wantNotPullRequest, err)
			}
		})
	}

	t.Run("a malformed pull request number is a real failure", func(t *testing.T) {
		t.Setenv("GITHUB_PULL_REQUEST_NUMBER", "not-a-number")
		t.Setenv("GITHUB_REF", "")
		t.Setenv("GITHUB_HEAD_REF", "")
		_, err := mg.DetectCurrentPullRequestNumber(ctx, "owner", "repo")
		if err == nil {
			t.Fatal("want err")
		}
		if errors.Is(err, ErrNotPullRequest) {
			t.Errorf("a malformed number must not read as an absent pull request (err: %v)", err)
		}
	})
}

func TestDeleteArtifactsBeforeRun(t *testing.T) {
	// Only the artifacts earlier runs uploaded are deleted. A later run can finish first, and
	// an artifact that names no run cannot be shown to be older.
	t.Setenv("GITHUB_TOKEN", "dummy")

	artifact := func(id, runID int64) *github.Artifact {
		a := &github.Artifact{ID: new(id), Name: new("octocov-report@refs_pull_1")}
		if runID != 0 {
			a.WorkflowRun = &github.ArtifactWorkflowRun{ID: new(runID)}
		}
		return a
	}
	var deleted []int64
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatchPages( //nostyle:funcfmt
			mock.GetReposActionsArtifactsByOwnerByRepo,
			github.ArtifactList{
				Artifacts: []*github.Artifact{artifact(1, 30), artifact(2, 20)},
			},
			github.ArtifactList{
				Artifacts: []*github.Artifact{
					artifact(3, 10),
					artifact(4, 0),
					{ID: github.Int64(5), Name: new("octocov-report@refs_pull_10"), WorkflowRun: &github.ArtifactWorkflowRun{ID: github.Int64(10)}},
				},
			},
		),
		mock.WithRequestMatchHandler( //nostyle:funcfmt
			mock.DeleteReposActionsArtifactsByOwnerByRepoByArtifactId,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
				if err != nil {
					t.Error(err)
				}
				deleted = append(deleted, id)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	if err := g.DeleteArtifactsBeforeRun(t.Context(), "owner", "repo", "octocov-report@refs_pull_1", 20); err != nil {
		t.Fatal(err)
	}
	want := []int64{3}
	if diff := cmp.Diff(deleted, want, nil); diff != "" {
		t.Error(diff)
	}
}

func TestDeleteArtifactsBeforeRunFails(t *testing.T) {
	// The error of a delete the token is not allowed to make is returned, so the caller can say
	// why the previous reports are still there.
	t.Setenv("GITHUB_TOKEN", "dummy")

	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposActionsArtifactsByOwnerByRepo,
			github.ArtifactList{
				Artifacts: []*github.Artifact{
					{ID: github.Int64(1), Name: new("octocov-report@refs_pull_1"), WorkflowRun: &github.ArtifactWorkflowRun{ID: github.Int64(10)}},
				},
			},
		),
		mock.WithRequestMatchHandler( //nostyle:funcfmt
			mock.DeleteReposActionsArtifactsByOwnerByRepoByArtifactId,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mock.WriteError(w, http.StatusForbidden, "Resource not accessible by integration")
			}),
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	if err := g.DeleteArtifactsBeforeRun(t.Context(), "owner", "repo", "octocov-report@refs_pull_1", 20); err == nil {
		t.Error("want err")
	}
}

func TestArtifactURL(t *testing.T) {
	tests := []struct {
		serverURL string
		want      string
	}{
		{"", "https://github.com/owner/repo/actions/runs/10/artifacts/20"},
		{"https://github.com/", "https://github.com/owner/repo/actions/runs/10/artifacts/20"},
		{"https://github.example.com", "https://github.example.com/owner/repo/actions/runs/10/artifacts/20"},
	}
	for _, tt := range tests {
		t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
		if got := ArtifactURL("owner", "repo", 10, 20); got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestFetchMergeBase(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "dummy")
	var basehead string
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			github.PullRequest{
				Base: &github.PullRequestBranch{SHA: new("base")},
				Head: &github.PullRequestBranch{SHA: new("head")},
			},
		),
		mock.WithRequestMatchHandler( //nostyle:funcfmt
			mock.GetReposCompareByOwnerByRepoByBasehead,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				basehead = path.Base(r.URL.Path)
				_, _ = w.Write(mock.MustMarshal(github.CommitsComparison{ //nostyle:handlerrors
					MergeBaseCommit: &github.RepositoryCommit{SHA: new("mergebase")},
				}))
			}),
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	got, err := g.FetchMergeBase(t.Context(), "owner", "repo", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "mergebase" {
		t.Errorf("got %v\nwant %v", got, "mergebase")
	}
	if want := "base...head"; basehead != want {
		t.Errorf("got %v\nwant %v", basehead, want)
	}
}

func TestFindRunArtifactReadsEveryPage(t *testing.T) {
	// A run with a large matrix holds more artifacts than one page does, and the one looked
	// for can be on any of them.
	t.Setenv("GITHUB_TOKEN", "dummy")
	named := func(id int64, name string) *github.Artifact {
		return &github.Artifact{ID: new(id), Name: new(name)}
	}
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatchPages( //nostyle:funcfmt
			mock.GetReposActionsRunsArtifactsByOwnerByRepoByRunId,
			github.ArtifactList{Artifacts: []*github.Artifact{named(1, "a"), named(2, "b")}},
			github.ArtifactList{Artifacts: []*github.Artifact{named(3, "octocov-report@refs_pull_1.html")}},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	a, err := g.findRunArtifact(t.Context(), "owner", "repo", 10, "octocov-report@refs_pull_1.html")
	if err != nil {
		t.Fatal(err)
	}
	if a == nil || a.GetID() != 3 {
		t.Errorf("got %v\nwant the artifact on the second page", a)
	}
}

func TestFetchChangedFilesCarryWhatTheFileCardsDraw(t *testing.T) {
	// A run whose pull request cannot be looked up falls back to these, and the page draws
	// each of them as a card with its status and its patch.
	t.Setenv("GITHUB_TOKEN", "dummy")
	t.Setenv("GITHUB_HEAD_REF", "feature")
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposByOwnerByRepo,
			github.Repository{DefaultBranch: new("main")},
		),
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposCompareByOwnerByRepoByBasehead,
			github.CommitsComparison{Files: []*github.CommitFile{{
				Filename: new("a.go"), Status: new("added"), Additions: new(1),
				Patch: new("@@ -0,0 +1 @@\n+x"),
			}}},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	files, err := g.FetchChangedFiles(t.Context(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	want := []*PullRequestFile{{Filename: "a.go", Status: "added", Additions: 1, Patch: "@@ -0,0 +1 @@\n+x", ChangedLines: []int{1}}}
	if diff := cmp.Diff(files, want); diff != "" {
		t.Error(diff)
	}
}

func TestDeleteRunArtifacts(t *testing.T) {
	// The pages earlier attempts of a re-run uploaded are on any page of the run's listing,
	// and only the ones the match names are deleted.
	t.Setenv("GITHUB_TOKEN", "dummy")
	named := func(id int64, name string) *github.Artifact {
		return &github.Artifact{ID: new(id), Name: new(name)}
	}
	var deleted []int64
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatchPages( //nostyle:funcfmt
			mock.GetReposActionsRunsArtifactsByOwnerByRepoByRunId,
			github.ArtifactList{Artifacts: []*github.Artifact{named(1, "page.html"), named(2, "report")}},
			github.ArtifactList{Artifacts: []*github.Artifact{named(3, "page@attempt_2.html")}},
		),
		mock.WithRequestMatchHandler( //nostyle:funcfmt
			mock.DeleteReposActionsArtifactsByOwnerByRepoByArtifactId,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
				if err != nil {
					t.Error(err)
				}
				deleted = append(deleted, id)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	if err := g.DeleteRunArtifacts(t.Context(), "owner", "repo", 10, func(name string) bool {
		return strings.HasPrefix(name, "page")
	}); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(deleted, []int64{1, 3}); diff != "" {
		t.Error(diff)
	}
}

func TestDeleteCompletedArtifactsBeforeRun(t *testing.T) {
	// An earlier run still in progress may yet link to the page it uploaded, so only the
	// pages of the runs that have completed are deleted.
	t.Setenv("GITHUB_TOKEN", "dummy")
	page := func(id, runID int64) *github.Artifact {
		return &github.Artifact{ID: new(id), Name: new("page.html"), WorkflowRun: &github.ArtifactWorkflowRun{ID: new(runID)}}
	}
	var deleted []int64
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposActionsArtifactsByOwnerByRepo,
			github.ArtifactList{Artifacts: []*github.Artifact{page(1, 10), page(2, 20), page(3, 10)}},
		),
		mock.WithRequestMatchHandler( //nostyle:funcfmt
			mock.GetReposActionsRunsByOwnerByRepoByRunId,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status := "completed"
				if path.Base(r.URL.Path) == "20" {
					status = "in_progress"
				}
				if _, err := w.Write(mock.MustMarshal(github.WorkflowRun{Status: new(status)})); err != nil {
					t.Error(err)
				}
			}),
		),
		mock.WithRequestMatchHandler( //nostyle:funcfmt
			mock.DeleteReposActionsArtifactsByOwnerByRepoByArtifactId,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
				if err != nil {
					t.Error(err)
				}
				deleted = append(deleted, id)
				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	if err := g.DeleteCompletedArtifactsBeforeRun(t.Context(), "owner", "repo", "page.html", 30); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(deleted, []int64{1, 3}); diff != "" {
		t.Error(diff)
	}
}

func TestHasReports(t *testing.T) {
	// A report an earlier run wrote is told by its signature, which is the key's.
	t.Setenv("GITHUB_TOKEN", "dummy")
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposIssuesCommentsByOwnerByRepoByIssueNumber,
			[]*github.IssueComment{{ID: new(int64(1)), Body: new("report\n<!-- octocov:sub -->")}},
			[]*github.IssueComment{{ID: new(int64(1)), Body: new("report\n<!-- octocov:sub -->")}},
		),
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposPullsByOwnerByRepoByPullNumber,
			github.PullRequest{Body: new("text\n<!-- octocov -->\nreport\n<!-- octocov -->")},
			github.PullRequest{Body: new("text\n<!-- octocov -->\nreport\n<!-- octocov -->")},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	g, err := New()
	if err != nil {
		t.Fatal(err)
	}
	g.SetClient(client)

	for _, tt := range []struct {
		name string
		has  func(key string) (bool, error)
		key  string
		want bool
	}{
		{"a comment of the key", func(k string) (bool, error) { return g.HasCommentReport(t.Context(), "owner", "repo", 1, k) }, "sub", true},
		{"a comment of another key", func(k string) (bool, error) { return g.HasCommentReport(t.Context(), "owner", "repo", 1, k) }, "", false},
		{"a body of the key", func(k string) (bool, error) { return g.HasBodyReport(t.Context(), "owner", "repo", 1, k) }, "", true},
		{"a body of another key", func(k string) (bool, error) { return g.HasBodyReport(t.Context(), "owner", "repo", 1, k) }, "sub", false},
	} {
		got, err := tt.has(tt.key)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFetchLatestArtifactOfBranch(t *testing.T) {
	// Every ref shares the artifact name, so only the branch the run was on tells the report of
	// the base apart from the ones pull requests stored under the same name later.
	t.Setenv("GITHUB_TOKEN", "dummy")
	zips := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		zw := zip.NewWriter(buf)
		f, err := zw.Create("report.json")
		if err != nil {
			t.Error(err)
		}
		if _, err := f.Write([]byte(path.Base(r.URL.Path))); err != nil {
			t.Error(err)
		}
		if err := zw.Close(); err != nil {
			t.Error(err)
		}
		_, _ = w.Write(buf.Bytes()) //nostyle:handlerrors
	}))
	t.Cleanup(zips.Close)

	artifact := func(id int64, branch string) *github.Artifact {
		return &github.Artifact{
			ID:          new(id),
			Name:        new("octocov-report"),
			WorkflowRun: &github.ArtifactWorkflowRun{HeadBranch: new(branch)},
		}
	}
	tests := []struct {
		name    string
		pages   []github.ArtifactList
		branch  string
		want    string
		wantErr error
	}{
		{
			"the newest artifact of the branch, past a newer one of a pull request",
			[]github.ArtifactList{{Artifacts: []*github.Artifact{artifact(3, "feat"), artifact(2, "main"), artifact(1, "main")}}},
			"main", "2", nil,
		},
		{
			"a branch whose artifacts are on a later page",
			[]github.ArtifactList{
				{Artifacts: []*github.Artifact{artifact(4, "feat"), artifact(3, "feat")}},
				{Artifacts: []*github.Artifact{artifact(2, "main")}},
			},
			"main", "2", nil,
		},
		{
			"no artifact of the branch",
			[]github.ArtifactList{{Artifacts: []*github.Artifact{artifact(2, "feat"), artifact(1, "fix")}}},
			"main", "", ErrArtifactNotFound,
		},
		{
			"an empty branch takes the newest of any",
			[]github.ArtifactList{{Artifacts: []*github.Artifact{artifact(3, "feat"), artifact(2, "main")}}},
			"", "3", nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := make([]any, 0, len(tt.pages))
			for _, p := range tt.pages {
				pages = append(pages, p)
			}
			mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
				mock.WithRequestMatchPages(mock.GetReposActionsArtifactsByOwnerByRepo, pages...),
				mock.WithRequestMatchHandler( //nostyle:funcfmt
					mock.GetReposActionsArtifactsByOwnerByRepoByArtifactIdByArchiveFormat,
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						id, err := strconv.ParseInt(path.Base(path.Dir(r.URL.Path)), 10, 64)
						if err != nil {
							t.Error(err)
						}
						w.Header().Set("Location", fmt.Sprintf("%s/%d", zips.URL, id))
						w.WriteHeader(http.StatusFound)
					}),
				),
			)
			client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient), factory.Timeout(10*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			g, err := New()
			if err != nil {
				t.Fatal(err)
			}
			g.SetClient(client)

			got, err := g.FetchLatestArtifactOfBranch(t.Context(), "owner", "repo", "octocov-report", "report.json", tt.branch)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v\nwant %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got.Content) != tt.want {
				t.Errorf("got %v\nwant %v", string(got.Content), tt.want)
			}
		})
	}
}
