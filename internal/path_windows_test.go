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
		{"C:\\path\\to", "C:\\path\\to", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"github.com\\owner\\repo\\foo\\file.txt"}, "github.com\\owner\\repo"},
		{"C:\\path\\to", "C:\\path\\to\\foo", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"github.com\\owner\\repo\\foo\\file.txt"}, "github.com\\owner\\repo\\foo"},
		{"C:\\path\\to", "C:\\path\\to\\bar", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"github.com\\owner\\repo\\foo\\file.txt"}, "github.com\\owner\\repo\\bar"},
		{"C:\\path\\a\\b\\c\\owner\\repo", "C:\\path\\a\\b\\c\\owner\\repo\\foo", []string{"C:\\path\\a\\b\\c\\owner\\repo\\foo\\bar\\bar.txt", "C:\\path\\a\\b\\c\\owner\\repo\\foo\\one\\two.txt"}, []string{"github.com\\owner\\repo\\foo\\bar\\bar.txt", "github.com\\owner\\repo\\foo\\one\\two.txt"}, "github.com\\owner\\repo\\foo"},
		{"C:\\path\\to", "C:\\path\\to", []string{"C:\\path\\to\\central\\central.go"}, []string{"github.com\\owner\\repo\\central\\central.go"}, "github.com\\owner\\repo"},
		{"C:\\path\\to\\github.com\\owner\\repo", "C:\\path\\to\\github.com\\owner\\repo", []string{"C:\\path\\to\\github.com\\owner\\repo\\central\\central.go"}, []string{"github.com\\owner\\repo\\central\\central.go"}, "github.com\\owner\\repo"},
		{"C:\\path\\to", "C:\\path\\to", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"C:\\other\\to\\foo\\file.txt"}, "C:\\other\\to"},
		{"C:\\path\\to", "C:\\path\\to", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"C:\\path\\to\\foo\\file.txt"}, "C:\\path\\to"},
		{"C:\\path\\to", "C:\\path\\to", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"C:\\path\\to\\bar\\foo\\file.txt"}, "C:\\path\\to\\bar"},
		{"C:\\path\\to", "C:\\path\\to\\foo", []string{"C:\\path\\to\\foo\\file.txt"}, []string{"C:\\path\\to\\bar\\foo\\file.txt"}, "C:\\path\\to\\bar\\foo"},
		{"C:\\path\\to", "C:\\path\\to\\foo", []string{"C:\\path\\to\\foo\\file.txt"}, []string{".\\foo\\file.txt"}, "foo"},
		{"C:\\path\\to", "C:\\path\\to", []string{"C:\\path\\to\\foo\\file.txt"}, []string{".\\foo\\file.txt"}, ""},
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
