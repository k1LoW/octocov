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

func TestLcovAcceptsPathContainingColon(t *testing.T) {
	// Windows drive letters put a ':' inside the SF value.
	dir := t.TempDir()
	path := filepath.Join(dir, "lcov.info")
	content := `TN:
SF:C:\proj\src\a.ts
DA:1,1
end_of_record
TN:
SF:C:\proj\src\b.ts
DA:1,0
end_of_record
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 2; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if want := `C:\proj\src\a.ts`; got.Files[0].File != want {
		t.Errorf("got %v\nwant %v", got.Files[0].File, want)
	}
	if want := `C:\proj\src\b.ts`; got.Files[1].File != want {
		t.Errorf("got %v\nwant %v", got.Files[1].File, want)
	}
	if want := 2; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 1; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
}

func TestLcovAcceptsChecksum(t *testing.T) {
	// DA:<line>,<count>[,<checksum>] (lcov --checksum)
	dir := t.TempDir()
	path := filepath.Join(dir, "lcov.info")
	content := `TN:
SF:src/a.ts
DA:1,1,abcdef
DA:2,0,abcdef
end_of_record
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 2; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 1; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
}

func TestLcovMergesRecordsOfSameFile(t *testing.T) {
	// Concatenated .info files (e.g. one per test run) repeat SF for the same
	// source file. The file must appear once, with counts stacked per line.
	dir := t.TempDir()
	path := filepath.Join(dir, "lcov.info")
	content := `TN:
SF:src/a.ts
DA:1,1
DA:2,0
end_of_record
TN:
SF:src/a.ts
DA:1,0
DA:2,3
DA:3,1
end_of_record
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	f := got.Files[0]
	if want := 3; f.Total != want {
		t.Errorf("got %v\nwant %v", f.Total, want)
	}
	if want := 3; f.Covered != want {
		t.Errorf("got %v\nwant %v", f.Covered, want)
	}
	if want := 5; len(f.Blocks) != want {
		t.Errorf("got %v\nwant %v", len(f.Blocks), want)
	}
	if want := 3; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 3; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
	// Exclude() re-sums the totals the parser folded and refolds only when Blocks changed.
	// The totals must not change.
	if err := got.Exclude(nil); err != nil {
		t.Fatal(err)
	}
	if want := 3; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 3; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
}

func TestLcovCountsRepeatedLineInOneRecordOnce(t *testing.T) {
	// A single record listing the same line twice (e.g. a hand-assembled or
	// concatenated .info) must count that line once, and a line hit by any of
	// its DA: entries must read as covered.
	dir := t.TempDir()
	path := filepath.Join(dir, "lcov.info")
	content := `TN:
SF:src/a.ts
DA:1,1
DA:1,0
DA:2,1
end_of_record
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	f := got.Files[0]
	if want := 2; f.Total != want {
		t.Errorf("got %v\nwant %v", f.Total, want)
	}
	if want := 2; f.Covered != want {
		t.Errorf("got %v\nwant %v", f.Covered, want)
	}
	// The blocks keep every listed line, so the totals came from folding them rather than
	// from an append that dropped the repeat.
	if want := 3; len(f.Blocks) != want {
		t.Errorf("got %v\nwant %v", len(f.Blocks), want)
	}
	if want := 2; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 2; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
	// Exclude() re-sums the totals the parser folded and refolds only when Blocks changed.
	// The totals must not change.
	if err := got.Exclude(nil); err != nil {
		t.Fatal(err)
	}
	if want := 2; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 2; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
}

func TestLcovBranches(t *testing.T) {
	// Written out here rather than kept under testdata, where the **/*.info patterns of the
	// report tests would pick it up as one more report.
	path := filepath.Join(t.TempDir(), "lcov.info")
	content := `TN:
SF:src/calc.ts
FN:1,add
FNDA:1,add
FNF:1
FNH:1
DA:1,1
DA:2,1
DA:3,1
DA:4,0
DA:6,0
DA:7,0
LF:6
LH:3
BRDA:2,0,0,1
BRDA:2,0,1,0
BRDA:3,1,0,2
BRDA:3,1,1,1
BRDA:3,2,0,0
BRDA:6,3,0,-
BRDA:6,3,1,-
BRF:7
BRH:3
end_of_record
TN:
SF:src/util.ts
DA:1,1
DA:2,1
LF:2
LH:2
BRDA:2,e0,0,1
BRDA:2,e0,1,0
BRF:2
BRH:1
end_of_record
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := [2]int{got.BranchTotal, got.BranchCovered}, [2]int{9, 4}; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	// The BRDA lines of src/calc.ts on line 6 read "-", as the block holding them never ran.
	want := map[string][2]int{
		"src/calc.ts": {7, 3},
		"src/util.ts": {2, 1},
	}
	for _, f := range got.Files {
		if got, want := [2]int{f.BranchTotal, f.BranchCovered}, want[f.File]; got != want {
			t.Errorf("%s: got %v\nwant %v", f.File, got, want)
		}
	}
}

func TestLcovWithoutBranches(t *testing.T) {
	path := filepath.Join(testdataDir(t), "lcov")
	got, _, err := NewLcov().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := [2]int{got.BranchTotal, got.BranchCovered}, [2]int{0, 0}; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestLcovBranchesOfRecords(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    [2]int
	}{
		{
			"branches of one line add up within a record",
			`SF:src/a.ts
BRDA:1,0,0,1
BRDA:1,0,1,0
BRDA:1,1,0,3
end_of_record
`,
			[2]int{3, 2},
		},
		{
			"a branch listed twice in a record counts once",
			`SF:src/a.ts
BRDA:1,0,0,0
BRDA:1,0,1,0
BRDA:1,0,0,1
end_of_record
`,
			[2]int{2, 1},
		},
		{
			"a line observed by two records counts once, as covered as the better of them",
			`SF:src/a.ts
BRDA:1,0,0,1
BRDA:1,0,1,0
BRDA:2,1,0,0
BRDA:2,1,1,0
end_of_record
SF:src/a.ts
BRDA:1,0,0,0
BRDA:1,0,1,1
BRDA:2,1,0,0
BRDA:2,1,1,2
end_of_record
`,
			[2]int{4, 2},
		},
		{
			"a branch whose block never ran is not covered",
			`SF:src/a.ts
BRDA:1,0,0,-
BRDA:1,0,1,-
end_of_record
`,
			[2]int{2, 0},
		},
		{
			"an unreadable line is skipped",
			`SF:src/a.ts
BRDA:1,0,0,1
BRDA:x,0,1,1
BRDA:1,0,1,y
BRDA:1,0
end_of_record
`,
			[2]int{1, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lcov.info")
			if err := os.WriteFile(path, []byte(tt.content), 0600); err != nil {
				t.Fatal(err)
			}
			got, _, err := NewLcov().ParseReport(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := [2]int{got.BranchTotal, got.BranchCovered}; got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
			// Exclude() refolds the branches, and must reach the same totals.
			if err := got.Exclude(nil); err != nil {
				t.Fatal(err)
			}
			if got := [2]int{got.BranchTotal, got.BranchCovered}; got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}
