package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRootPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b", "c", "d"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "a", "b", ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "a", "b", ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	tests := []struct {
		base    string
		wantErr bool
	}{
		{filepath.Join(dir, "a", "b", "c"), false},
		{filepath.Join(dir, "a", "b", "c", "d"), false},
		{filepath.Join(dir, "a", "b"), false},
		{filepath.Join(dir, "a"), true},
	}
	for _, tt := range tests {
		got, err := GetRootPath(tt.base)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		} else {
			if tt.wantErr {
				t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
			}
			if want := filepath.Join(dir, "a", "b"); got != want {
				t.Errorf("got %v\nwant %v", got, want)
			}
		}
	}
}

func TestDetectPrefix(t *testing.T) {
	tests := []struct {
		gitRoot string
		wd      string
		files   []string
		cfiles  []string
		want    string
	}{
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"github.com/owner/repo/foo/file.txt"}, "github.com/owner/repo"},
		{"/path/to", "/path/to/foo", []string{"/path/to/foo/file.txt"}, []string{"github.com/owner/repo/foo/file.txt"}, "github.com/owner/repo/foo"},
		{"/path/to", "/path/to/bar", []string{"/path/to/foo/file.txt"}, []string{"github.com/owner/repo/foo/file.txt"}, "github.com/owner/repo/bar"},
		{"/path/to", "/path/to", []string{"/path/to/central/central.go"}, []string{"github.com/owner/repo/central/central.go"}, "github.com/owner/repo"},
		{"/path/to/github.com/owner/repo", "/path/to/github.com/owner/repo", []string{"/path/to/github.com/owner/repo/central/central.go"}, []string{"github.com/owner/repo/central/central.go"}, "github.com/owner/repo"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"/other/to/foo/file.txt"}, "/other/to"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"/path/to/foo/file.txt"}, "/path/to"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"/path/to/bar/foo/file.txt"}, "/path/to/bar"},
		{"/path/to", "/path/to/foo", []string{"/path/to/foo/file.txt"}, []string{"/path/to/bar/foo/file.txt"}, "/path/to/bar/foo"},
	}
	for _, tt := range tests {
		got := DetectPrefix(tt.gitRoot, tt.wd, tt.files, tt.cfiles)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}
