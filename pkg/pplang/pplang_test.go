package pplang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v45/github"
	"github.com/josharian/txtarfs"
	"github.com/k1LoW/go-github-client/v45/factory"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"golang.org/x/tools/txtar"
)

func TestDetect(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Dir(filepath.Dir(wd)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Detect(dir)
	if err != nil {
		t.Error(err)
	}
	if want := "Go"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestDetectFS(t *testing.T) {
	tests := []struct {
		txtar string
		want  string
	}{
		{"go.txtar", "Go"},
	}
	for _, tt := range tests {
		a, err := txtar.ParseFile(filepath.Join(testdataDir(t), tt.txtar))
		if err != nil {
			t.Error(err)
			continue
		}
		got, err := DetectFS(txtarfs.As(a))
		if err != nil {
			t.Error(err)
			continue
		}
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestDetectUsingAPI(t *testing.T) {
	tests := []struct {
		env   string
		txtar string
		want  string
	}{
		{"owner/repo", "none.txtar", "Ruby"},
		{"", "none.txtar", ""},
		{"", "gitconfig.txtar", "Ruby"},
	}
	for _, tt := range tests {
		t.Setenv("GITHUB_REPOSITORY", tt.env)
		a, err := txtar.ParseFile(filepath.Join(testdataDir(t), tt.txtar))
		if err != nil {
			t.Error(err)
			continue
		}
		got, _ := DetectUsingAPI(mockedClient(t), txtarfs.As(a))
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func mockedClient(t *testing.T) *github.Client {
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposLanguagesByOwnerByRepo,
			map[string]int{
				"Ruby":       12345,
				"JavaScript": 12344,
			},
		),
	)
	client, err := factory.NewGithubClient(factory.HTTPClient(mockedHTTPClient))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(wd, "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
