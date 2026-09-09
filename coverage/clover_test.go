package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClover(t *testing.T) {
	path := filepath.Join(testdataDir(t), "clover")
	clover := NewClover()
	got, _, err := clover.ParseReport(path)
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

func TestCloverPackage(t *testing.T) {
	path := filepath.Join(testdataDir(t), "clover", "coverage_package.xml")
	clover := NewClover()
	got, _, err := clover.ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	cover := false
	for _, f := range got.Files {
		if f.File == "/path/to/src/app/libs/Util.php" {
			cover = true
		}
	}
	if !cover {
		t.Error("does not parse <package> section")
	}
}

func TestCloverPathAttribute(t *testing.T) {
	path := filepath.Join(testdataDir(t), "clover", "coverage_path.xml")
	clover := NewClover()
	got, _, err := clover.ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Files) != 3 {
		t.Fatalf("got %d files, want 3", len(got.Files))
	}

	wantFiles := map[string]struct {
		total   int
		covered int
	}{
		"/src/components/utils.ts": {total: 10, covered: 8},
		"/src/helpers/utils.ts":    {total: 10, covered: 6},
		"/src/lib/utils.ts":        {total: 10, covered: 6},
	}

	for _, f := range got.Files {
		want, ok := wantFiles[f.File]
		if !ok {
			t.Errorf("unexpected file: %s", f.File)
			continue
		}
		if f.Total != want.total {
			t.Errorf("%s: total got %d, want %d", f.File, f.Total, want.total)
		}
		if f.Covered != want.covered {
			t.Errorf("%s: covered got %d, want %d", f.File, f.Covered, want.covered)
		}
		delete(wantFiles, f.File)
	}
	for f := range wantFiles {
		t.Errorf("missing file: %s", f)
	}
}

func TestCloverParseAllFormat(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{filepath.Join(testdataDir(t), "gocover", "coverage.out"), true},
		{filepath.Join(testdataDir(t), "lcov", "lcov.info"), true},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json"), true},
		{filepath.Join(testdataDir(t), "clover", "coverage.xml"), false},
		{filepath.Join(testdataDir(t), "cobertura", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "jacoco", "jacocoTestReport.xml"), true},
	}
	for _, tt := range tests {
		_, _, err := NewClover().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
}

func TestCloverCountsFromLines(t *testing.T) {
	// The totals come from the <line type="stmt"> elements, folded per line, and not from the
	// <metrics> attributes, so what ParseReport returns is what its blocks support and what
	// Exclude recounts to. The fixture declares statements="36" coveredstatements="28" for a
	// file listing ten executed statement lines, and statements="0" for one listing a single
	// executed line.
	path := filepath.Join(testdataDir(t), "clover", "coverage_package.xml")
	got, _, err := NewClover().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		file    string
		total   int
		covered int
	}{
		{"/path/to/src/Framework/Exception.php", 0, 0},
		{"/path/to/src/Framework/Assert.php", 10, 10},
		{"/path/to/src/app/libs/Util.php", 1, 1},
	}
	for _, tt := range tests {
		f, err := got.Files.FindByFile(tt.file)
		if err != nil {
			t.Fatal(err)
		}
		if f.Total != tt.total || f.Covered != tt.covered {
			t.Errorf("%s: got %d/%d\nwant %d/%d", tt.file, f.Total, f.Covered, tt.total, tt.covered)
		}
	}
	if want := 11; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 11; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
	// Exclude() recalculates from blocks; the totals must not change.
	if err := got.Exclude(nil); err != nil {
		t.Fatal(err)
	}
	if got.Total != 11 || got.Covered != 11 {
		t.Errorf("got %d/%d\nwant 11/11", got.Total, got.Covered)
	}
}

func parseCloverString(t *testing.T, content string) *Coverage {
	t.Helper()
	path := filepath.Join(t.TempDir(), "coverage.xml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewClover().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestCloverMergesFileNamedTwice(t *testing.T) {
	// A report can describe one file at the project level and again inside a <package>. When
	// both elements carry the path attribute the file is listed once, with the blocks of both
	// stacked and counted once per line. The report below describes one file whose single
	// listed line was executed.
	got := parseCloverString(t, `<?xml version="1.0" ?>
<coverage generated="1">
  <project timestamp="1">
    <file name="a.php" path="/src/a.php">
      <metrics statements="5" coveredstatements="4"/>
      <line num="1" type="stmt" count="1"/>
    </file>
    <package name="pkg">
      <file name="a.php" path="/src/a.php">
        <metrics statements="5" coveredstatements="4"/>
        <line num="1" type="stmt" count="1"/>
      </file>
    </package>
  </project>
</coverage>
`)
	if want := 1; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if want := "/src/a.php"; got.Files[0].File != want {
		t.Errorf("got %v\nwant %v", got.Files[0].File, want)
	}
	if got.Total != 1 || got.Covered != 1 {
		t.Errorf("got %d/%d\nwant 1/1", got.Total, got.Covered)
	}
	// The blocks keep every listed line, so the totals came from folding them rather than
	// from an append that dropped the repeat.
	if want := 2; len(got.Files[0].Blocks) != want {
		t.Errorf("got %v\nwant %v", len(got.Files[0].Blocks), want)
	}
	if err := got.Exclude(nil); err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 || got.Covered != 1 {
		t.Errorf("after Exclude got %d/%d\nwant 1/1", got.Total, got.Covered)
	}
}

func TestCloverMergesAbsoluteNameWithoutPath(t *testing.T) {
	// An absolute name is a real location on its own, so two elements naming it merge without a
	// path attribute, which is how PHPUnit writes its reports.
	got := parseCloverString(t, `<?xml version="1.0" ?>
<coverage generated="1">
  <project timestamp="1">
    <file name="/src/a.php">
      <line num="1" type="stmt" count="1"/>
    </file>
    <package name="pkg">
      <file name="/src/a.php">
        <line num="2" type="stmt" count="0"/>
      </file>
    </package>
  </project>
</coverage>
`)
	if want := 1; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if got.Total != 2 || got.Covered != 1 {
		t.Errorf("got %d/%d\nwant 2/1", got.Total, got.Covered)
	}
}

func TestCloverKeepsSameNamedFilesWithoutPathApart(t *testing.T) {
	// Files in different packages can share a bare name, and without a path attribute that name
	// is all the report gives, so it cannot say whether two such elements are one file or two.
	// They stay two entries. The report below has two files of two lines, one fully covered and
	// one not at all.
	got := parseCloverString(t, `<?xml version="1.0" ?>
<coverage generated="1">
  <project timestamp="1">
    <package name="src/a">
      <file name="utils.ts">
        <line num="1" type="stmt" count="1"/>
        <line num="2" type="stmt" count="1"/>
      </file>
    </package>
    <package name="src/b">
      <file name="utils.ts">
        <line num="1" type="stmt" count="0"/>
        <line num="2" type="stmt" count="0"/>
      </file>
    </package>
  </project>
</coverage>
`)
	if want := 2; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if got.Total != 4 || got.Covered != 2 {
		t.Errorf("got %d/%d\nwant 4/2", got.Total, got.Covered)
	}
}

func TestIsAbsReportPath(t *testing.T) {
	// The answer must not depend on the host octocov runs on, since the path describes the host
	// that produced the report.
	tests := []struct {
		path string
		want bool
	}{
		{"/src/a.php", true},
		{`C:\src\a.php`, true},
		{"c:/src/a.php", true},
		{"src/a.php", false},
		{"a.php", false},
		{"./a.php", false},
		{"", false},
		{"C:", false},
		{`:\a.php`, false},
		{`1:\a.php`, false},
	}
	for _, tt := range tests {
		if got := isAbsReportPath(tt.path); got != tt.want {
			t.Errorf("%q: got %v\nwant %v", tt.path, got, tt.want)
		}
	}
}
