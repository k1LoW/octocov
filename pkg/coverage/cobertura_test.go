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
}
