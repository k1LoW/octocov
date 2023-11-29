package coverage

import (
	"path/filepath"
	"testing"
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
