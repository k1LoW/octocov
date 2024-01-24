//go:build !windows

package internal

import (
	"fmt"
	"testing"
)

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
		{"/path/a/b/c/owner/repo", "/path/a/b/c/owner/repo/foo", []string{"/path/a/b/c/owner/repo/foo/bar/bar.txt", "/path/a/b/c/owner/repo/foo/one/two.txt"}, []string{"github.com/owner/repo/foo/bar/bar.txt", "github.com/owner/repo/foo/one/two.txt"}, "github.com/owner/repo/foo"},
		{"/path/to", "/path/to", []string{"/path/to/central/central.go"}, []string{"github.com/owner/repo/central/central.go"}, "github.com/owner/repo"},
		{"/path/to/github.com/owner/repo", "/path/to/github.com/owner/repo", []string{"/path/to/github.com/owner/repo/central/central.go"}, []string{"github.com/owner/repo/central/central.go"}, "github.com/owner/repo"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"/other/to/foo/file.txt"}, "/other/to"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"/path/to/foo/file.txt"}, "/path/to"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"/path/to/bar/foo/file.txt"}, "/path/to/bar"},
		{"/path/to", "/path/to/foo", []string{"/path/to/foo/file.txt"}, []string{"/path/to/bar/foo/file.txt"}, "/path/to/bar/foo"},
		{"/path/to", "/path/to/foo", []string{"/path/to/foo/file.txt"}, []string{"./foo/file.txt"}, "foo"},
		{"/path/to", "/path/to", []string{"/path/to/foo/file.txt"}, []string{"./foo/file.txt"}, ""},
		// {"/path/a/b/c/github.com/owner/repo/", "/path/a/b/c/github.com/owner/repo/foo", []string{"/path/a/b/c/github.com/owner/repo/bar/config/config.go", "/path/a/b/c/github.com/owner/repo/foo/config/config.go", "/path/a/b/c/github.com/owner/repo/foo/entity/hello.go"}, []string{"github.com/owner/repo/foo/config/config.go", "github.com/owner/repo/foo/entity/hello.go"}, "github.com/owner/repo/foo"},
	}
	for i, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Parallel()
			got := DetectPrefix(tt.gitRoot, tt.wd, tt.files, tt.cfiles)
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}
