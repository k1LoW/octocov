package coverage

import (
	"path/filepath"
	"testing"
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
	}
	for _, tt := range tests {
		_, _, err := NewCobertura().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
}
