package coverage

import (
	"os"
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
	if want := "./lib/ridgepole.rb"; got.Files[0].File != want {
		t.Errorf("got %v\nwant %v", got.Files[0].File, want)
	}

	for _, f := range got.Files {
		total := 0
		covered := 0
		for _, b := range f.Blocks {
			// LOC
			total = total + 1
			if *b.Count > 0 {
				covered += 1
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

func TestLcovAcceptsU64WrappedCounts(t *testing.T) {
	// llvm-cov (e.g. cargo-llvm-cov) can emit u64-wrapped negative execution
	// counts when profile counters race; one such line must not reject the
	// whole report, and the raw u64 value must survive parsing.
	dir := t.TempDir()
	path := filepath.Join(dir, "lcov.info")
	content := `TN:
SF:src/cache.rs
DA:1,1
DA:2,18446744073709551611
DA:3,0
LF:3
LH:2
end_of_record
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 3; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	// The wrapped counter still counts as executed.
	if want := 2; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
	if want := ExecCount(18446744073709551611); *got.Files[0].Blocks[1].Count != want {
		t.Errorf("got %v\nwant %v", *got.Files[0].Blocks[1].Count, want)
	}
}

func TestLcovParseAllFormat(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{filepath.Join(testdataDir(t), "gocover", "coverage.out"), true},
		{filepath.Join(testdataDir(t), "lcov", "lcov.info"), false},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json"), true},
		{filepath.Join(testdataDir(t), "clover", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "cobertura", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "jacoco", "jacocoTestReport.xml"), true},
	}
	for _, tt := range tests {
		_, _, err := NewLcov().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
}
