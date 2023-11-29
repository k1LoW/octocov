package pplang

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v50/github"
	"github.com/josharian/txtarfs"
	"github.com/k1LoW/go-github-client/v50/factory"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"golang.org/x/tools/txtar"
)

func TestDetect(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Dir(wd))
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
		env     string
		txtar   string
		want    string
		wantErr bool
	}{
		{"owner/repo", "none.txtar", "Ruby", false},
		{"", "none.txtar", "", true},
		{"", "gitconfig.txtar", "Ruby", false},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", tt.env)
			a, err := txtar.ParseFile(filepath.Join(testdataDir(t), tt.txtar))
			if err != nil {
				t.Error(err)
				return
			}
			got, err := DetectUsingAPI(mockedClient(t), txtarfs.As(a))
			if err != nil {
				if !tt.wantErr {
					t.Error(err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want error")
				return
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func mockedClient(t *testing.T) *github.Client {
	mockedHTTPClient := mock.NewMockedHTTPClient( //nostyle:funcfmt
		mock.WithRequestMatch( //nostyle:funcfmt
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
