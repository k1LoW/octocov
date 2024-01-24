//go:build !windows

package datastore

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseUNIX(t *testing.T) {
	var tests = []struct {
		in        string
		wantType  Type
		wantArgs  []string
		wantError bool
	}{
		{"file://reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"file:///reports", Local, []string{"/reports"}, false},
		{"/reports", Local, []string{"/reports"}, false},
		{"local://reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"local://./reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"local:///reports", Local, []string{"/reports"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			gotType, gotArgs, err := parse(tt.in, testdataDir(t))
			if err != nil {
				if !tt.wantError {
					t.Errorf("got %v", err)
				}
				return
			}
			if err == nil && tt.wantError {
				t.Error("want error")
			}
			if gotType != tt.wantType {
				t.Errorf("got %v\nwant %v", gotType, tt.wantType)
			}
			if diff := cmp.Diff(gotArgs, tt.wantArgs, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}
