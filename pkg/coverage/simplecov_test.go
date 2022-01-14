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
}
