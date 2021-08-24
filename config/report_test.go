package config

import (
	"testing"
)

func TestReportConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want bool
	}{
		{
			New(),
			false,
		},
		{
			&Config{
				Report: &ConfigReport{
					Datastores: []string{
						"s3://octocov-test/reports",
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		got := tt.c.ReportConfigReady()
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}
