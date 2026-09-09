package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCobertura(t *testing.T) {
	path := filepath.Join(testdataDir(t), "cobertura")
	cobertura := NewCobertura()
	got, _, err := cobertura.ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := 7712; got.Total != want {
		t.Errorf("got %v\nwant %v", got.Total, want)
	}
	if want := 7706; got.Covered != want {
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

func TestCoberturaParseAllFormat(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{filepath.Join(testdataDir(t), "gocover", "coverage.out"), true},
		{filepath.Join(testdataDir(t), "lcov", "lcov.info"), true},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json"), true},
		{filepath.Join(testdataDir(t), "clover", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "cobertura", "coverage.xml"), false},
		{filepath.Join(testdataDir(t), "jacoco", "jacocoTestReport.xml"), true},
	}
	for _, tt := range tests {
		_, _, err := NewCobertura().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
}

func TestCoberturaFilesOrder(t *testing.T) {
	path := filepath.Join(testdataDir(t), "cobertura")
	// The files of a parsed report follow the order the report lists them in, and stay in that
	// order however many times the same report is parsed. Sorted order would put
	// dependencies/__init__.py sixth, so the sixth name is what tells the two apart.
	head := []string{
		"__init__.py",
		"applications.py",
		"background.py",
		"concurrency.py",
		"datastructures.py",
		"encoders.py",
	}
	var first []string
	for i := range 10 {
		cov, _, err := NewCobertura().ParseReport(path)
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

func TestCoberturaMergesClassesOfSameFile(t *testing.T) {
	// A file split over several <class> elements (one per inner class, say) appears once, at
	// the position of its first element, with the lines of every element behind it.
	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.xml")
	content := `<?xml version="1.0" ?>
<coverage>
  <packages>
    <package name="pkg">
      <classes>
        <class filename="a.py">
          <lines><line number="1" hits="1"/></lines>
        </class>
        <class filename="b.py">
          <lines><line number="1" hits="0"/></lines>
        </class>
        <class filename="a.py">
          <lines><line number="2" hits="3"/></lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewCobertura().ParseReport(path)
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, f := range got.Files {
		files = append(files, f.File)
	}
	if diff := cmp.Diff(files, []string{"a.py", "b.py"}); diff != "" {
		t.Fatal(diff)
	}
	if want := 2; len(got.Files[0].Blocks) != want {
		t.Errorf("got %v\nwant %v", len(got.Files[0].Blocks), want)
	}
}

func TestCoberturaCountsSharedLineOnce(t *testing.T) {
	// A line listed under two <class> elements of one file counts once, and counts as covered
	// when any of those elements records a hit. The file below has three distinct lines, of
	// which 1 and 2 were executed, and line 1 is listed under both classes.
	dir := t.TempDir()
	path := filepath.Join(dir, "coverage.xml")
	content := `<?xml version="1.0" ?>
<coverage>
  <packages>
    <package name="com.example">
      <classes>
        <class filename="com/example/Foo.kt" name="Foo">
          <lines>
            <line number="1" hits="1"/>
            <line number="2" hits="1"/>
          </lines>
        </class>
        <class filename="com/example/Foo.kt" name="Foo.bar.1">
          <lines>
            <line number="1" hits="0"/>
            <line number="3" hits="0"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	got, _, err := NewCobertura().ParseReport(path)
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
