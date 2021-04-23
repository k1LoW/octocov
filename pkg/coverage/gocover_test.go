package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGocover(t *testing.T) {
	path := filepath.Join(testdataDir(t), "gocover")
	gcov := NewGocover()
	got, err := gcov.ParseReport(path)
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
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(filepath.Dir(wd)), "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
