package config

import (
	"testing"
)

func TestReportConfigReady(t *testing.T) {
	tests := []struct {
		c       *Config
		wantErr bool
	}{
		{
			New(),
			true,
		},
		{
			&Config{
				Report: &ConfigReport{
					Datastores: []string{
						"s3://octocov-test/reports",
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		if err := tt.c.ReportConfigReady(); err != nil {
			if !tt.wantErr {
				t.Errorf("got error %v\n", err)
			}
		} else {
			if tt.wantErr {
				t.Error("want error\n")
			}
		}
	}
}
