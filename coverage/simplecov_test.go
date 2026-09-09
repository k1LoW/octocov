package coverage

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSimplecov(t *testing.T) {
	tests := []struct {
		path string
	}{
		{filepath.Join(testdataDir(t), "simplecov")},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json")},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset2.json")},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.another.json")},
	}
	for _, tt := range tests {
		scov := NewSimplecov()
		got, _, err := scov.ParseReport(tt.path)
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
}

func TestTotalAndCovered(t *testing.T) {
	tests := []struct {
		pathA string
		pathB string
	}{
		{
			filepath.Join(testdataDir(t), "simplecov"),
			filepath.Join(testdataDir(t), "simplecov", ".resultset.json"),
		},
		{
			filepath.Join(testdataDir(t), "simplecov", ".resultset.json"),
			filepath.Join(testdataDir(t), "simplecov", ".resultset.another.json"),
		},
		{
			filepath.Join(testdataDir(t), "simplecov", ".resultset.json"),
			filepath.Join(testdataDir(t), "simplecov", ".resultset.parallel.json"),
		},
	}
	for _, tt := range tests {
		gotA, _, err := NewSimplecov().ParseReport(tt.pathA)
		if err != nil {
			t.Fatal(err)
		}

		gotB, _, err := NewSimplecov().ParseReport(tt.pathB)
		if err != nil {
			t.Fatal(err)
		}

		if gotA.Total != gotB.Total {
			t.Errorf("gotA %v\ngotB %v", gotA.Total, gotB.Total)
		}

		if gotA.Covered != gotB.Covered {
			t.Errorf("gotA %v\ngotB %v", gotA.Covered, gotB.Covered)
		}
	}
}

func TestSimplecovParseAllFormat(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{filepath.Join(testdataDir(t), "gocover", "coverage.out"), true},
		{filepath.Join(testdataDir(t), "lcov", "lcov.info"), true},
		{filepath.Join(testdataDir(t), "simplecov", ".resultset.json"), false},
		{filepath.Join(testdataDir(t), "clover", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "cobertura", "coverage.xml"), true},
		{filepath.Join(testdataDir(t), "jacoco", "jacocoTestReport.xml"), true},
	}
	for _, tt := range tests {
		_, _, err := NewSimplecov().ParseReport(tt.path)
		if tt.wantErr != (err != nil) {
			t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
		}
	}
}

func TestSimplecovFilesOrder(t *testing.T) {
	path := filepath.Join(testdataDir(t), "simplecov")
	// A .resultset.json is a JSON object, so it fixes no order for its files. They come out
	// sorted by name, the same order however many times the same report is parsed.
	var first []string
	for i := range 10 {
		cov, _, err := NewSimplecov().ParseReport(path)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, f := range cov.Files {
			got = append(got, f.File)
		}
		if diff := cmp.Diff(got, slices.Sorted(slices.Values(got))); diff != "" {
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
