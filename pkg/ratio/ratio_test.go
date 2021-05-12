package ratio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasure(t *testing.T) {
	tests := []struct {
		code    []string
		test    []string
		wantErr bool
	}{
		{[]string{}, []string{}, false},
		{[]string{"**/*.go", "!**/*_test.go"}, []string{"**/*_test.go"}, false},
		{[]string{"**/*.ts"}, []string{}, true},
	}
	for _, tt := range tests {
		root := filepath.Join(testdataDir(t), "..")
		got, err := Measure(root, tt.code, tt.test)
		if err != nil {
			if !tt.wantErr {
				t.Error(err)
			}
		} else {
			if tt.wantErr {
				t.Errorf("got %v\nwant err", got)
			}
		}
	}
}

func TestPathMatch(t *testing.T) {
	root := filepath.Join(testdataDir(t), "..")
	{
		code := []string{
			"**/*.go",
			"!**/*_test.go",
		}
		got, err := Measure(root, code, []string{})
		if err != nil {
			t.Fatal(err)
		}
		if contains(got.CodeFiles, "pkg/ratio/ratio_test.go") {
			t.Error("pkg/ratio/ratio_test.go should not be contained")
		}
	}

	{
		code := []string{
			"!**/*_test.go",
			"**/*.go",
		}
		got, err := Measure(root, code, []string{})
		if err != nil {
			t.Fatal(err)
		}
		if !contains(got.CodeFiles, "pkg/ratio/ratio_test.go") {
			t.Error("pkg/ratio/ratio_test.go should be contained")
		}
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(filepath.Dir(wd)), "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
