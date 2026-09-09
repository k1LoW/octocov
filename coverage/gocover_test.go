package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGocover(t *testing.T) {
	path := filepath.Join(testdataDir(t), "gocover")
	gcov := NewGocover()
	got, _, err := gcov.ParseReport(path)
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
	for _, f := range got.Files {
		total := 0
		covered := 0
		for _, b := range f.Blocks {
			// Statement
			total = total + *b.NumStmt
			if *b.Count > 0 {
				covered += *b.NumStmt
			}
		}
		if got := f.Total; got != total {
			t.Errorf("got %v\nwant %v", got, total)
		}
		if got := f.Covered; got != covered {
			t.Errorf("got %v\nwant %v", got, covered)
		}
	}
}

func TestGocoverParseAllFormat(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{filepath.Join(testdataDir(t), "gocover", "coverage.out"), false},
		{filepath.Join(testdataDir(t), "lcov", "lcov.info"), true},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json"), true},
		{filepath.Join(testdataDir(t), "clover", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "cobertura", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "jacoco", "jacocoTestReport.xml"), true},
	}
	for _, tt := range tests {
		_, _, err := NewGocover().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
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

func TestGocoverRejectsImplausibleBlock(t *testing.T) {
	// x/tools/cover accepts any integer for a block's lines and columns, and Go cover is the one
	// format whose blocks span lines, so a profile can describe a run of source no file has. The
	// parse must fail rather than hand the block to a walk that allocates per line.
	tests := []struct {
		name    string
		block   string
		wantErr bool
	}{
		{"wide line span", "x.go:1.1,9223372036854775806.1 1 1", true},
		{"wide column span", "x.go:1.1,1.9223372036854775806 1 1", true},
		{"ends before it starts", "x.go:9.1,3.1 1 1", true},
		{"furthest line accepted", "x.go:1.1,1000000.1 1 1", false},
		{"narrow block past the bound", "x.go:1000001.1,1000002.1 1 1", true},
		// The usual shape of a block that spans lines, ending at a column below where it began.
		{"spans lines", "x.go:10.66,12.25 1 1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "coverage.out")
			if err := os.WriteFile(path, []byte("mode: count\n"+tt.block+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, _, err := NewGocover().ParseReport(path)
			if tt.wantErr != (err != nil) {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		})
	}
}
