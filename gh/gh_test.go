package gh

import (
	"context"
	"errors"
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
	got, err := mg.FetchDefaultBranch(context.TODO(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}

	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestFetchRawRootURL(t *testing.T) {
	ctx := context.TODO()
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
		{"refs/heads/name", "mybranch", "name", false},
		{"refs/heads/branch/branch/name", "", "branch/branch/name", false},
		{"refs/pull/8/head", "mybranch", "mybranch", false},
	}
	ctx := context.TODO()
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
		GITHUB_PULL_REQUEST_NUMBER string
		GITHUB_REF                 string
		want                       int
		wantErr                    bool
	}{
		{"", "refs/pull/8/head", 8, false},
		{"", "refs/heads/branch/branch/name", 13, false},
		{"", "refs/8", 0, true},
		{"8", "", 8, false},
		{"str", "", 0, true},
	}
	ctx := context.TODO()
	mg := mockedGh(t)
	for _, tt := range tests {
		t.Run(tt.GITHUB_REF, func(t *testing.T) {
			t.Setenv("GITHUB_PULL_REQUEST_NUMBER", tt.GITHUB_PULL_REQUEST_NUMBER)
			t.Setenv("GITHUB_REF", tt.GITHUB_REF)
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
				DefaultBranch: github.String("main"),
			},
		),
		mock.WithRequestMatch( //nostyle:funcfmt
			mock.GetReposPullsByOwnerByRepo,
			[]*github.PullRequest{
				{
					Head: &github.PullRequestBranch{
						Ref: github.String("branch/branch/name"),
						Repo: &github.Repository{
							Owner: &github.User{Login: github.String("owner")},
							Name:  github.String("repo"),
						},
					},
					Number: github.Int(13),
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
					Number: github.Int(10),
					Head: &github.PullRequestBranch{
						Ref: github.String("feature-branch"),
						Repo: &github.Repository{
							Owner: &github.User{Login: github.String("owner")},
							Name:  github.String("repo"),
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
					Number: github.Int(20),
					Head: &github.PullRequestBranch{
						Ref: github.String("main"),
						Repo: &github.Repository{
							Owner: &github.User{Login: github.String("forked-user")},
							Name:  github.String("repo"),
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
					Number: github.Int(20),
					Head: &github.PullRequestBranch{
						Ref: github.String("main"),
						Repo: &github.Repository{
							Owner: &github.User{Login: github.String("forked-user")},
							Name:  github.String("repo"),
						},
					},
				},
				{
					Number: github.Int(30),
					Head: &github.PullRequestBranch{
						Ref: github.String("main"),
						Repo: &github.Repository{
							Owner: &github.User{Login: github.String("owner")},
							Name:  github.String("repo"),
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
					Number: github.Int(25),
					Head: &github.PullRequestBranch{
						Ref: github.String("feature"),
						Repo: &github.Repository{
							Owner: &github.User{Login: github.String("Owner")},
							Name:  github.String("repo"),
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
					Number: github.Int(35),
					Head: &github.PullRequestBranch{
						Ref:  github.String("feature"),
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

			got, err := g.DetectCurrentPullRequestNumber(context.TODO(), "owner", "repo")
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
				TotalCount: github.Int(3),
				Jobs: []*github.WorkflowJob{
					{ID: github.Int64(1), Name: github.String("test (1)")},
					{ID: github.Int64(2), Name: github.String("test (2)")},
				},
			},
			github.Jobs{
				TotalCount: github.Int(3),
				Jobs: []*github.WorkflowJob{
					{ID: github.Int64(3), Name: github.String("test (3)")},
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

	jobs, err := g.listWorkflowJobs(context.TODO(), "owner", "repo", 1)
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
						Name:        github.String("Run test"),
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
						Name:      github.String("Run test"),
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
					github.Jobs{TotalCount: github.Int(len(tt.jobs)), Jobs: tt.jobs},
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
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
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
				Owner: &github.User{Login: github.String("owner")},
				Name:  github.String("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  true,
		},
		{
			name: "owner case insensitive",
			headRepo: &github.Repository{
				Owner: &github.User{Login: github.String("Owner")},
				Name:  github.String("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  true,
		},
		{
			name: "different owner",
			headRepo: &github.Repository{
				Owner: &github.User{Login: github.String("forked-user")},
				Name:  github.String("repo"),
			},
			owner: "owner",
			repo:  "repo",
			want:  false,
		},
		{
			name: "different repo name",
			headRepo: &github.Repository{
				Owner: &github.User{Login: github.String("owner")},
				Name:  github.String("other-repo"),
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
				Name:  github.String("repo"),
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
	ctx := context.TODO()
	mg := mockedGh(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_PULL_REQUEST_NUMBER", "")
			t.Setenv("GITHUB_REF", tt.GITHUB_REF)
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
		_, err := mg.DetectCurrentPullRequestNumber(ctx, "owner", "repo")
		if err == nil {
			t.Fatal("want err")
		}
		if errors.Is(err, ErrNotPullRequest) {
			t.Errorf("a malformed number must not read as an absent pull request (err: %v)", err)
		}
	})
}
