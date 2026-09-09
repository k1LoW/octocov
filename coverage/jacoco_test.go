package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestJacoco(t *testing.T) {
	path := filepath.Join(testdataDir(t), "jacoco")
	Jacoco := NewJacoco()
	got, _, err := Jacoco.ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 11219; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 10351; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
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

func TestJacocoParseAllFormat(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{filepath.Join(testdataDir(t), "gocover", "coverage.out"), true},
		{filepath.Join(testdataDir(t), "lcov", "lcov.info"), true},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json"), true},
		{filepath.Join(testdataDir(t), "clover", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "cobertura", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "jacoco", "jacocoTestReport.xml"), false},
	}
	for _, tt := range tests {
		_, _, err := NewJacoco().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
}

func TestJacocoFilesOrder(t *testing.T) {
	path := filepath.Join(testdataDir(t), "jacoco")
	// The files of a parsed report follow the order the report lists them in, and stay in that
	// order however many times the same report is parsed. Sorted order would start at
	// io/cloudevents/cloudEventsExtensions.kt, so these two names rule sorting out.
	head := []string{
		"org/http4k/security/oauth/server/accesstoken/GenerateAccessTokenForGrantType.kt",
		"org/http4k/security/oauth/server/accesstoken/GrantConfiguration.kt",
	}
	var first []string
	for i := range 10 {
		cov, _, err := NewJacoco().ParseReport(path)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, f := range cov.Files {
			got = append(got, f.File)
		}
		if len(got) < len(head) {
			t.Fatalf("got %v files\nwant at least %v", len(got), len(head))
		}
		if diff := cmp.Diff(got[:len(head)], head); diff != "" {
			t.Error(diff)
		}
		if i == 0 {
			first = got
			continue
		}
		if diff := cmp.Diff(got, first); diff != "" {
			t.Error(diff)
		}
	}
}

func TestJacocoMergesPackagesOfSameName(t *testing.T) {
	// A report aggregated from several modules can carry the same <package> name twice. The
	// files it names appear once, at the position of their first element, with the lines of
	// every element behind them.
	dir := t.TempDir()
	path := filepath.Join(dir, "jacocoTestReport.xml")
	content := `<?xml version="1.0" ?>
<report name="agg">
  <package name="org/example">
    <sourcefile name="A.kt">
      <line nr="1" mi="0" ci="1"/>
    </sourcefile>
    <sourcefile name="B.kt">
      <line nr="1" mi="1" ci="0"/>
    </sourcefile>
  </package>
  <package name="org/example">
    <sourcefile name="A.kt">
      <line nr="2" mi="0" ci="3"/>
    </sourcefile>
  </package>
</report>
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewJacoco().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, f := range got.Files {
		files = append(files, f.File)
	}
	if diff := cmp.Diff(files, []string{"org/example/A.kt", "org/example/B.kt"}); diff != "" {
		t.Fatal(diff)
	}
	if want := 2; len(got.Files[0].Blocks) != want {
		t.Errorf("got %v\nwant %v", len(got.Files[0].Blocks), want)
	}
}

func TestJacocoCountsSharedLineOnce(t *testing.T) {
	// A repeated <package> name can bring the same file back with a line it already listed.
	// That line counts once, and counts as covered when any of the elements records a hit. The
	// file below has three distinct lines, of which 1 and 2 were executed.
	dir := t.TempDir()
	path := filepath.Join(dir, "jacocoTestReport.xml")
	content := `<?xml version="1.0" ?>
<report name="agg">
  <package name="org/example">
    <sourcefile name="A.kt">
      <line nr="1" mi="0" ci="1"/>
      <line nr="2" mi="0" ci="1"/>
    </sourcefile>
  </package>
  <package name="org/example">
    <sourcefile name="A.kt">
      <line nr="1" mi="1" ci="0"/>
      <line nr="3" mi="1" ci="0"/>
    </sourcefile>
  </package>
</report>
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewJacoco().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 3; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 2; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
	if want := 1; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if want := 3; got.Files[0].Total != want {
		t.Errorf("got %v\nwant %v", got.Files[0].Total, want)
	}
	if want := 2; got.Files[0].Covered != want {
		t.Errorf("got %v\nwant %v", got.Files[0].Covered, want)
	}
	// The blocks keep every listed line, so the totals came from folding them rather than
	// from an append that dropped the repeat.
	if want := 4; len(got.Files[0].Blocks) != want {
		t.Errorf("got %v\nwant %v", len(got.Files[0].Blocks), want)
	}
}

func TestJacocoNamesDefaultPackageFileRelative(t *testing.T) {
	// A class in the default package has <package name="">. Its file must be named without a
	// leading slash, so that FuzzyFindByFile can resolve it from the path it lives at in the
	// repository, which is the lookup the pull request scope table performs.
	dir := t.TempDir()
	path := filepath.Join(dir, "jacocoTestReport.xml")
	content := `<?xml version="1.0" ?>
<report name="t">
  <package name="">
    <sourcefile name="Foo.java">
      <line nr="1" mi="0" ci="1"/>
      <line nr="2" mi="1" ci="0"/>
    </sourcefile>
  </package>
  <package name="com/example">
    <sourcefile name="Bar.java">
      <line nr="1" mi="0" ci="1"/>
    </sourcefile>
    <sourcefile name="Foo.java">
      <line nr="1" mi="0" ci="1"/>
    </sourcefile>
  </package>
</report>
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewJacoco().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 3; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if want := "Foo.java"; got.Files[0].File != want {
		t.Errorf("got %v\nwant %v", got.Files[0].File, want)
	}
	for _, f := range got.Files {
		if filepath.IsAbs(f.File) {
			t.Errorf("got absolute path %v", f.File)
		}
	}
	// Each file resolves from the path it lives at, and the same basename in the default
	// package and in a named package go to their own files.
	tests := []struct {
		file string
		want string
	}{
		{"src/main/java/Foo.java", "Foo.java"},
		{"src/main/java/com/example/Bar.java", "com/example/Bar.java"},
		{"src/main/java/com/example/Foo.java", "com/example/Foo.java"},
	}
	for _, tt := range tests {
		f, err := got.Files.FuzzyFindByFile(tt.file)
		if err != nil {
			t.Errorf("%s: %v", tt.file, err)
			continue
		}
		if f.File != tt.want {
			t.Errorf("got %v\nwant %v", f.File, tt.want)
		}
	}
}

func TestJacocoSkipsSourcefileWithoutName(t *testing.T) {
	// A <sourcefile> with no name would key an entry on "" in the default package and on
	// "pkg/" in a named one. Neither can resolve a changed path, so none is made and the named
	// file beside them is unaffected.
	dir := t.TempDir()
	path := filepath.Join(dir, "jacocoTestReport.xml")
	content := `<?xml version="1.0" ?>
<report name="t">
  <package name="">
    <sourcefile>
      <line nr="1" mi="0" ci="1"/>
    </sourcefile>
    <sourcefile name="Foo.java">
      <line nr="1" mi="1" ci="0"/>
    </sourcefile>
  </package>
  <package name="com/example">
    <sourcefile>
      <line nr="1" mi="0" ci="1"/>
    </sourcefile>
  </package>
</report>
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewJacoco().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 1; len(got.Files) != want {
		t.Fatalf("got %v\nwant %v", len(got.Files), want)
	}
	if want := "Foo.java"; got.Files[0].File != want {
		t.Errorf("got %v\nwant %v", got.Files[0].File, want)
	}
	if want := 0; got.Covered != want {
		t.Errorf("got %v\nwant %v", got.Covered, want)
	}
}
