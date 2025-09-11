package report

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tenntenn/golden"
	"golang.org/x/text/language"
)

func TestCustomMetricSetTable(t *testing.T) {
	tests := []struct {
		s *CustomMetricSet
	}{
		{&CustomMetricSet{}},
		{&CustomMetricSet{
			Key:  "benchmark_0",
			Name: "Benchmark-0",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
			},
		}},
		{&CustomMetricSet{
			Key:  "benchmark_1",
			Name: "Benchmark-1",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1500.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 1340.0, Unit: " ns/op"},
			},
		}},
		{&CustomMetricSet{
			Key:  "many_metrics",
			Name: "Many Metrics",
			Metrics: []*CustomMetric{
				{Key: "A", Name: "Metrics A", Value: 1500.0, Unit: ""},
				{Key: "B", Name: "Metrics B", Value: 1340.0, Unit: ""},
				{Key: "C", Name: "Metrics C", Value: 1600.0, Unit: ""},
				{Key: "D", Name: "Metrics D", Value: 1010.0, Unit: ""},
				{Key: "E", Name: "Metrics E", Value: 1800.0, Unit: ""},
			},
			report: &Report{
				Ref:      "main",
				Commit:   "1234567890",
				covPaths: []string{"testdata/cover.out"},
			},
		}},
		{&CustomMetricSet{
			Key:  "benchmark_0",
			Name: "Benchmark-0",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
			},
			report: &Report{
				opts: &Options{Locale: &language.French},
			},
		}},
		{&CustomMetricSet{
			Key:  "benchmark_1",
			Name: "Benchmark-1",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1500.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 1340.0, Unit: " ns/op"},
			},
			report: &Report{
				opts: &Options{Locale: &language.Japanese},
			},
		}},
		{&CustomMetricSet{
			Key:  "many_metrics",
			Name: "Many Metrics",
			Metrics: []*CustomMetric{
				{Key: "A", Name: "Metrics A", Value: 1500.0, Unit: ""},
				{Key: "B", Name: "Metrics B", Value: 1340.0, Unit: ""},
				{Key: "C", Name: "Metrics C", Value: 1600.0, Unit: ""},
				{Key: "D", Name: "Metrics D", Value: 1010.0, Unit: ""},
				{Key: "E", Name: "Metrics E", Value: 1800.0, Unit: ""},
			},
			report: &Report{
				Ref:      "main",
				Commit:   "1234567890",
				covPaths: []string{"testdata/cover.out"},
				opts:     &Options{Locale: &language.French},
			},
		}},
	}
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := tt.s.Table()
			f := filepath.Join("custom_metrics", fmt.Sprintf("custom_metric_set_table.%d", i))
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, testdataDir(t), f, got)
				return
			}
			if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestCustomMetricSetMetadataTable(t *testing.T) {
	tests := []struct {
		s *CustomMetricSet
	}{
		{&CustomMetricSet{}},
		{&CustomMetricSet{
			Key:  "benchmark_0",
			Name: "Benchmark-0",
			Metadata: []*MetadataKV{
				{Key: "goos", Value: "darwin"},
				{Key: "goarch", Value: "amd64"},
				{Key: "pkg", Value: "github.com/k1LoW/octocov/metrics"},
				{Key: "commit", Value: "a1b2c3d4e5f6"},
			},
		}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := tt.s.MetadataTable()
			f := filepath.Join("custom_metrics", fmt.Sprintf("custom_metric_set_metadata_table.%d", i))
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, testdataDir(t), f, got)
				return
			}
			if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestCustomMetricSetOut(t *testing.T) {
	tests := []struct {
		s *CustomMetricSet
	}{
		{&CustomMetricSet{}},
		{&CustomMetricSet{
			Key:  "benchmark_0",
			Name: "Benchmark-0",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
			},
			report: &Report{
				Ref:      "main",
				Commit:   "1234567890",
				covPaths: []string{"testdata/cover.out"},
			},
		}},
		{&CustomMetricSet{
			Key:  "benchmark_1",
			Name: "Benchmark-1",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1500.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 1340.0, Unit: " ns/op"},
			},
			report: &Report{
				Ref:      "main",
				Commit:   "1234567890",
				covPaths: []string{"testdata/cover.out"},
			},
		}},
		{&CustomMetricSet{
			Key:  "benchmark_0",
			Name: "Benchmark-0",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
			},
			report: &Report{
				Ref:      "main",
				Commit:   "1234567890",
				covPaths: []string{"testdata/cover.out"},
				opts:     &Options{Locale: &language.French},
			},
		}},
		{&CustomMetricSet{
			Key:  "benchmark_1",
			Name: "Benchmark-1",
			Metrics: []*CustomMetric{
				{Key: "N", Name: "Number of iterations", Value: 1500.0, Unit: ""},
				{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 1340.0, Unit: " ns/op"},
			},
			report: &Report{
				Ref:      "main",
				Commit:   "1234567890",
				covPaths: []string{"testdata/cover.out"},
				opts:     &Options{Locale: &language.Japanese},
			},
		}},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got := new(bytes.Buffer)
			if err := tt.s.Out(got); err != nil {
				t.Fatal(err)
			}
			f := filepath.Join("custom_metrics", fmt.Sprintf("custom_metric_set_out.%d", i))
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, testdataDir(t), f, got)
				return
			}
			if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestCustomMetricsSetValidate(t *testing.T) {
	tests := []struct {
		in      *CustomMetricSet
		wantErr bool
	}{
		{&CustomMetricSet{}, true},
		{&CustomMetricSet{
			Key: "key",
			Metrics: []*CustomMetric{
				{Key: "count", Value: 1000.0},
				{Key: "ns_per_op", Value: 676.0, Unit: "ns/op"},
			},
		}, false},
		{&CustomMetricSet{
			Key:     "key",
			Metrics: []*CustomMetric{},
		}, true},
		{&CustomMetricSet{
			Key: "key",
			Metrics: []*CustomMetric{
				{Key: "count", Value: 1000.0},
				{Key: "count", Value: 1001.0},
			},
		}, true},
		{&CustomMetricSet{
			Key: "key",
			Metrics: []*CustomMetric{
				{Key: "count", Value: 1000.0},
			},
			Metadata: []*MetadataKV{
				{Key: "goos", Value: "darwin"},
				{Key: "goarch", Value: "amd64"},
			},
		}, false},
		{&CustomMetricSet{
			Key: "key",
			Metrics: []*CustomMetric{
				{Key: "count", Value: 1000.0},
			},
			Metadata: []*MetadataKV{
				{Key: "goos", Value: "darwin"},
				{Key: "goos", Value: "linux"},
			},
		}, true},
		{&CustomMetricSet{
			Key:         "key",
			Metrics:     []*CustomMetric{},
			Acceptables: []string{"true"},
		}, true},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if err := tt.in.Validate(); err != nil {
				if !tt.wantErr {
					t.Error(err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want error")
			}
		})
	}
}

func TestDiffCustomMetricSetTable(t *testing.T) {
	tests := []struct {
		a *CustomMetricSet
		b *CustomMetricSet
	}{
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metrics: []*CustomMetric{
					{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
					{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
				},
			},
			nil,
		},
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metrics: []*CustomMetric{
					{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
					{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
				},
			},
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metrics: []*CustomMetric{
					{Key: "N", Name: "Number of iterations", Value: 9393.0, Unit: ""},
					{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 456.0, Unit: " ns/op"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "2345678901",
					covPaths: []string{"testdata/cover.out"},
				},
			},
		},
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metrics: []*CustomMetric{
					{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
					{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
					opts:     &Options{Locale: &language.French},
				},
			},
			nil,
		},
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metrics: []*CustomMetric{
					{Key: "N", Name: "Number of iterations", Value: 1000.0, Unit: ""},
					{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 676.5, Unit: " ns/op"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
					opts:     &Options{Locale: &language.Japanese},
				},
			},
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metrics: []*CustomMetric{
					{Key: "N", Name: "Number of iterations", Value: 9393.0, Unit: ""},
					{Key: "NsPerOp", Name: "Nanoseconds per iteration", Value: 456.0, Unit: " ns/op"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "2345678901",
					covPaths: []string{"testdata/cover.out"},
				},
			},
		},
	}

	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			d := tt.a.Compare(tt.b)
			got := d.Table()
			f := filepath.Join("custom_metrics", fmt.Sprintf("diff_custom_metric_set_table.%d", i))
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, testdataDir(t), f, got)
				return
			}
			if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestDiffCustomMetricSetMetadataTable(t *testing.T) {
	tests := []struct {
		a *CustomMetricSet
		b *CustomMetricSet
	}{
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metadata: []*MetadataKV{
					{Key: "goos", Value: "darwin"},
					{Key: "goarch", Value: "amd64"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
				},
			},
			nil,
		},
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metadata: []*MetadataKV{
					{Key: "goos", Value: "darwin"},
					{Key: "goarch", Value: "amd64"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
				},
			},
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metadata: []*MetadataKV{
					{Key: "goos", Value: "arwin"},
					{Key: "goarch", Value: "arm64"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "2345678901",
					covPaths: []string{"testdata/cover.out"},
				},
			},
		},
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metadata: []*MetadataKV{
					{Key: "goos", Value: "darwin"},
					{Key: "goarch", Value: "amd64"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
					opts:     &Options{Locale: &language.French},
				},
			},
			nil,
		},
		{
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metadata: []*MetadataKV{
					{Key: "goos", Value: "darwin"},
					{Key: "goarch", Value: "amd64"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "1234567890",
					covPaths: []string{"testdata/cover.out"},
					opts:     &Options{Locale: &language.Japanese},
				},
			},
			&CustomMetricSet{
				Key:  "benchmark_0",
				Name: "Benchmark-0",
				Metadata: []*MetadataKV{
					{Key: "goos", Value: "arwin"},
					{Key: "goarch", Value: "arm64"},
				},
				report: &Report{
					Ref:      "main",
					Commit:   "2345678901",
					covPaths: []string{"testdata/cover.out"},
				},
			},
		},
	}

	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			d := tt.a.Compare(tt.b)
			got := d.MetadataTable()
			f := filepath.Join("custom_metrics", fmt.Sprintf("diff_custom_metric_set_metadata_table.%d", i))
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, testdataDir(t), f, got)
				return
			}
			if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReport_CustomMetricsAcceptable(t *testing.T) {
	tests := []struct {
		name     string
		current  *Report
		prev     *Report
		wantErr  bool
		errorMsg string
	}{
		// Basic current variable usage
		{
			name: "basic_current_usage",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "test_metrics",
						Metrics: []*CustomMetric{
							{Key: "score", Value: 85.5},
						},
						Acceptables: []string{"current.score > 80"},
					},
				},
			},
			prev:    &Report{CustomMetrics: []*CustomMetricSet{}},
			wantErr: false,
		},
		// Previous comparison
		{
			name: "prev_comparison",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "test_metrics",
						Metrics: []*CustomMetric{
							{Key: "requests", Value: 2000.0},
						},
						Acceptables: []string{"current.requests > prev.requests"},
					},
				},
			},
			prev: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "test_metrics",
						Metrics: []*CustomMetric{
							{Key: "requests", Value: 1800.0},
						},
					},
				},
			},
			wantErr: false,
		},
		// Diff variable usage
		{
			name: "diff_usage",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "test_metrics",
						Metrics: []*CustomMetric{
							{Key: "performance", Value: 95.0},
						},
						Acceptables: []string{"diff.performance > 0"},
					},
				},
			},
			prev: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "test_metrics",
						Metrics: []*CustomMetric{
							{Key: "performance", Value: 90.0},
						},
					},
				},
			},
			wantErr: false,
		},
		// Condition failure
		{
			name: "condition_failure",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key:         "test",
						Metrics:     []*CustomMetric{{Key: "score", Value: 75}},
						Acceptables: []string{"current.score >= 80"},
					},
				},
			},
			prev: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key:     "test",
						Metrics: []*CustomMetric{{Key: "score", Value: 85}},
					},
				},
			},
			wantErr:  true,
			errorMsg: "not acceptable condition",
		},
		// No previous data
		{
			name: "no_previous_data",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "test_metrics",
						Metrics: []*CustomMetric{
							{Key: "new_metric", Value: 100.0},
						},
						Acceptables: []string{"current.new_metric > 0 && prev.new_metric == 0"},
					},
				},
			},
			prev:    &Report{CustomMetrics: []*CustomMetricSet{}},
			wantErr: false,
		},
		// Real usage pattern - performance monitoring
		{
			name: "performance_monitoring",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "performance_metrics",
						Metrics: []*CustomMetric{
							{Key: "response_time", Value: 250.0},
							{Key: "throughput", Value: 1500.0},
							{Key: "error_rate", Value: 0.5},
						},
						Acceptables: []string{
							"current.response_time < 300",
							"current.throughput > 1000",
							"current.error_rate < 1.0",
						},
					},
				},
			},
			prev: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "performance_metrics",
						Metrics: []*CustomMetric{
							{Key: "response_time", Value: 280.0},
							{Key: "throughput", Value: 1400.0},
							{Key: "error_rate", Value: 0.3},
						},
					},
				},
			},
			wantErr: false,
		},
		// Real usage pattern - quality gate
		{
			name: "quality_gate",
			current: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "quality_metrics",
						Metrics: []*CustomMetric{
							{Key: "coverage", Value: 85.0},
							{Key: "complexity", Value: 7.5},
						},
						Acceptables: []string{
							"current.coverage >= 80",
							"current.complexity <= 8.0",
							"current.coverage >= prev.coverage",
						},
					},
				},
			},
			prev: &Report{
				CustomMetrics: []*CustomMetricSet{
					{
						Key: "quality_metrics",
						Metrics: []*CustomMetric{
							{Key: "coverage", Value: 83.0},
							{Key: "complexity", Value: 7.8},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.current.CustomMetricsAcceptable(tt.prev)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("got error = %v, want error containing %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("got error = %v, want nil", err)
				}
			}
		})
	}
}
