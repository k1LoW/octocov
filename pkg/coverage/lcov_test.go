package coverage

import (
	"path/filepath"
	"testing"
)

func TestLcov(t *testing.T) {
	path := filepath.Join(testdataDir(t), "lcov")
	lcov := NewLcov()
	got, _, err := lcov.ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total == 0 {
		t.Error("got 0 want > 0")
	}
	if got.Covered == 0 {
		t.Error("got 0 want > 0")
	}
	if len(got.Files) == 0 {
		t.Error("got 0 want > 0")
	}
	if want := "./lib/ridgepole.rb"; got.Files[0].FileName != want {
		t.Errorf("got %v\nwant %v", got.Files[0].FileName, want)
	}
}
