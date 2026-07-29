package gh

import (
	"context"
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

func TestJobsFindStepByTime(t *testing.T) {
	// Simulates a GitHub Actions matrix: the "test" step runs concurrently in
	// three separate jobs (one per shard), each finishing at a different time.
	job := func(id int64, name string, started, completed time.Time) *github.WorkflowJob {
		return &github.WorkflowJob{
			ID:   github.Int64(id),
			Name: github.String(name),
			Steps: []*github.TaskStep{
				{
					Name:        github.String("Run test"),
					StartedAt:   &github.Timestamp{Time: started},
					CompletedAt: &github.Timestamp{Time: completed},
				},
			},
		}
	}
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	jobs := &Jobs{jobs: &github.Jobs{Jobs: []*github.WorkflowJob{
		job(1, "test (1)", base, base.Add(10*time.Minute)),
		job(2, "test (2)", base, base.Add(20*time.Minute)),
		job(3, "test (3)", base, base.Add(30*time.Minute)),
	}}}

	tests := []struct {
		name   string
		t      time.Time
		jobIDs []int64
		want   Step
		wantOk bool
	}{
		{"matches step in first job", base.Add(5 * time.Minute), []int64{1, 2, 3}, Step{"Run test", base, base.Add(10 * time.Minute)}, true},
		{"matches step in a later job", base.Add(25 * time.Minute), []int64{1, 2, 3}, Step{"Run test", base, base.Add(30 * time.Minute)}, true},
		{"no step covers this time", base.Add(time.Hour), []int64{1, 2, 3}, Step{}, false},
		{"job excluded from jobIDs is ignored", base.Add(25 * time.Minute), []int64{1, 2}, Step{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := jobs.FindStepByTime(tt.jobIDs, tt.t)
			if ok != tt.wantOk {
				t.Fatalf("got ok=%v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestJobsFindStepsByName(t *testing.T) {
	job := func(id int64, steps ...*github.TaskStep) *github.WorkflowJob {
		return &github.WorkflowJob{ID: github.Int64(id), Steps: steps}
	}
	step := func(name string, started, completed *time.Time) *github.TaskStep {
		s := &github.TaskStep{Name: github.String(name)}
		if started != nil {
			s.StartedAt = &github.Timestamp{Time: *started}
		}
		if completed != nil {
			s.CompletedAt = &github.Timestamp{Time: *completed}
		}
		return s
	}
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := base.Add(10 * time.Minute)

	tests := []struct {
		name   string
		jobs   *Jobs
		jobIDs []int64
		step   string
		want   []Step
		wantOk bool
	}{
		{
			name: "collects the named step across every matched job",
			jobs: &Jobs{jobs: &github.Jobs{Jobs: []*github.WorkflowJob{
				job(1, step("Run test", &base, &end)),
				job(2, step("Run test", &base, &end)),
			}}},
			jobIDs: []int64{1, 2},
			step:   "Run test",
			want:   []Step{{"Run test", base, end}, {"Run test", base, end}},
			wantOk: true,
		},
		{
			name: "job excluded from jobIDs is ignored",
			jobs: &Jobs{jobs: &github.Jobs{Jobs: []*github.WorkflowJob{
				job(1, step("Run test", &base, &end)),
			}}},
			jobIDs: []int64{2},
			step:   "Run test",
			wantOk: false,
		},
		{
			name: "step not yet completed is not ready",
			jobs: &Jobs{jobs: &github.Jobs{Jobs: []*github.WorkflowJob{
				job(1, step("Run test", &base, nil)),
			}}},
			jobIDs: []int64{1},
			step:   "Run test",
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := tt.jobs.FindStepsByName(tt.jobIDs, tt.step)
			if ok != tt.wantOk {
				t.Fatalf("got ok=%v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestJobsResolveIDs(t *testing.T) {
	tests := []struct {
		name        string
		jobPatterns []string
		jobs        []*github.WorkflowJob
		want        []int64
	}{
		{
			name:        "no patterns falls back to the single job in the run",
			jobPatterns: nil,
			jobs: []*github.WorkflowJob{
				{ID: github.Int64(1), Name: github.String("test")},
			},
			want: []int64{1},
		},
		{
			name:        "glob pattern matches every matrix job but not unrelated jobs",
			jobPatterns: []string{"test (*)"},
			jobs: []*github.WorkflowJob{
				{ID: github.Int64(1), Name: github.String("test (1)")},
				{ID: github.Int64(2), Name: github.String("test (2)")},
				{ID: github.Int64(3), Name: github.String("lint")},
			},
			want: []int64{1, 2},
		},
		{
			name:        "no job matches the pattern",
			jobPatterns: []string{"nomatch"},
			jobs: []*github.WorkflowJob{
				{ID: github.Int64(1), Name: github.String("test")},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			js := &Jobs{jobs: &github.Jobs{Jobs: tt.jobs}}
			got, err := js.ResolveIDs(tt.jobPatterns)
			if err != nil {
				t.Fatalf("got err: %v", err)
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestListWorkflowJobsRequiresRunID(t *testing.T) {
	mg := mockedGh(t)
	t.Setenv("GITHUB_RUN_ID", "")
	if _, err := mg.ListWorkflowJobs(context.TODO(), "owner", "repo"); err == nil {
		t.Error("want err")
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
