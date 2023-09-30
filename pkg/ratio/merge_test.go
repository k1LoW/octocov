package ratio

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		r1   *Ratio
		r2   *Ratio
		want *Ratio
	}{
		{
			&Ratio{
				Code: 10,
				Test: 4,
				CodeFiles: Files{
					&File{Path: "file_a.go", Code: 7},
					&File{Path: "file_b.go", Code: 3},
				},
				TestFiles: Files{
					&File{Path: "file_a_test.go", Code: 4},
				},
			},
			&Ratio{
				Code: 7,
				Test: 2,
				CodeFiles: Files{
					&File{Path: "file_c.go", Code: 7},
				},
				TestFiles: Files{
					&File{Path: "file_c_test.go", Code: 2},
				},
			},
			&Ratio{
				Code: 17,
				Test: 6,
				CodeFiles: Files{
					&File{Path: "file_a.go", Code: 7},
					&File{Path: "file_b.go", Code: 3},
					&File{Path: "file_c.go", Code: 7},
				},
				TestFiles: Files{
					&File{Path: "file_a_test.go", Code: 4},
					&File{Path: "file_c_test.go", Code: 2},
				},
			},
		},
		{
			&Ratio{
				Code: 10,
				Test: 4,
				CodeFiles: Files{
					&File{Path: "file_a.go", Code: 7},
					&File{Path: "file_b.go", Code: 3},
				},
				TestFiles: Files{
					&File{Path: "file_a_test.go", Code: 4},
				},
			},
			&Ratio{
				Code: 7,
				Test: 2,
				CodeFiles: Files{
					&File{Path: "file_b.go", Code: 7},
				},
				TestFiles: Files{
					&File{Path: "file_c_test.go", Code: 2},
				},
			},
			&Ratio{
				Code: 14,
				Test: 6,
				CodeFiles: Files{
					&File{Path: "file_a.go", Code: 7},
					&File{Path: "file_b.go", Code: 7},
				},
				TestFiles: Files{
					&File{Path: "file_a_test.go", Code: 4},
					&File{Path: "file_c_test.go", Code: 2},
				},
			},
		},
	}
	for _, tt := range tests {
		if err := tt.r1.Merge(tt.r2); err != nil {
			t.Fatal(err)
		}
		got := tt.r1
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Error(diff)
		}
	}
}
