package coverage

import (
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
