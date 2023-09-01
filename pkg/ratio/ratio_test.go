package ratio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCompare(t *testing.T) {
	a := &Ratio{
		Code: 100,
		Test: 250,
	}
	tests := []struct {
		b    *Ratio
		want *DiffRatio
	}{
		{
			&Ratio{
				Code: 100,
				Test: 250,
			},
			&DiffRatio{
				A:    2.5,
				B:    2.5,
				Diff: 0.0,
			},
		},
		{
			nil,
			&DiffRatio{
				A:    2.5,
				B:    0.0,
				Diff: 2.5,
			},
		},
		{
			&Ratio{
				Code: 100,
				Test: 300,
			},
			&DiffRatio{
				A:    2.5,
				B:    3.0,
				Diff: -0.5,
			},
		},
	}
	for _, tt := range tests {
		got := a.Compare(tt.b)

		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(DiffRatio{}),
			cmpopts.IgnoreFields(DiffRatio{}, "RatioA", "RatioB"),
		}

		if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}

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
		test := []string{
			"!**/*.go",
			"**/*_test.go",
		}
		got, err := Measure(root, code, test)
		if err != nil {
			t.Fatal(err)
		}
		want := "pkg/ratio/ratio_test.go"
		ok := false
		for _, f := range got.CodeFiles {
			if f.Path == want {
				ok = true
			}
		}
		if ok {
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
		want := "pkg/ratio/ratio_test.go"
		ok := false
		for _, f := range got.CodeFiles {
			if f.Path == want {
				ok = true
			}
		}
		if !ok {
			t.Error("pkg/ratio/ratio_test.go should be contained")
		}
	}
}

func TestDeleteFiles(t *testing.T) {
	root := filepath.Join(testdataDir(t), "..")
	code := []string{
		"**/*.go",
		"!**/*_test.go",
	}
	got, err := Measure(root, code, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.CodeFiles) == 0 {
		t.Errorf("got %v\nwant >0", len(got.CodeFiles))
	}
	got.DeleteFiles()
	if len(got.CodeFiles) > 0 {
		t.Errorf("got %v\nwant 0", len(got.CodeFiles))
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
