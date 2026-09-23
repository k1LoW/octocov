package condition

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEval(t *testing.T) {
	vars := map[string]any{
		"current": 80.5,
		"prev":    79.0,
		"hour":    9,
		"month":   time.September,
		"labels":  []string{"bug"},
		"metrics": map[string]float64{"a": 3},
		"env":     map[string]string{"GITHUB_REF": "refs/heads/main"},
		"github": map[string]any{
			"event": map[string]any{"number": 12.0, "action": "opened"},
		},
		"is_pull_request": true,
	}
	tests := []struct {
		cond       string
		want       bool
		wantErr    bool
		deprecated bool
	}{
		{"current >= 80", true, false, false},
		{"current >= 80.5", true, false, false},
		{"current > prev + 1", true, false, false},
		{"current - prev >= 2", false, false, false},
		{"current * 2 > 160", true, false, false},
		{"7 / 2 == 3.5", true, false, false},
		{"hour == 9", true, false, false},
		{"hour + 1 == 10", true, false, false},
		{"month == 9", true, false, false},
		{"'bug' in labels", true, false, false},
		{"size(labels) > 0", true, false, false},
		{"labels[0] == 'bug'", true, false, false},
		{"7 % 2 == 1", true, false, false},
		{"size(labels) == 1", true, false, false},
		{"size(labels) == 1 && current > prev + 1", true, false, false},
		{"hour % 2 == 1", true, false, false},
		{"labels[hour - 9] == 'bug'", true, false, false},
		{"current > -1", true, false, false},
		{"1 + 1 < current", true, false, false},
		{"metrics.a + 1 > 3", true, false, false},
		{"env.GITHUB_REF == 'refs/heads/main'", true, false, false},
		{"env[?'FOO'].orValue('') == ''", true, false, false},
		{"has(env.FOO) && env.FOO == 'x'", false, false, false},
		{"github.event.number + 1 == 13", true, false, false},
		{"is_pull_request && current > prev", true, false, false},

		// Conditions written for expr-lang/expr
		{"is_pull_request and current > prev", true, false, true},
		{"not is_pull_request", false, false, true},
		{"len(labels) == 1", true, false, true},
		{"env.FOO == 'x'", false, false, true},
		{"'refs/heads/main' startsWith 'refs/'", true, false, true},

		{"current", false, false, true},
		{"current >=", false, true, false},
		{"unknown > 1", false, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.cond, func(t *testing.T) {
			buf := captureWarnings(t)
			got, err := Eval(tt.cond, vars)
			if (err != nil) != tt.wantErr {
				t.Fatalf("got error %v, want error %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if deprecated := strings.Contains(buf.String(), "Deprecated"); deprecated != tt.deprecated {
				t.Errorf("got warning %q, want warning %v", buf.String(), tt.deprecated)
			}
		})
	}
}

func TestProgramWarnsOnce(t *testing.T) {
	buf := captureWarnings(t)
	p, err := Compile("current >= 80 and current < 90", map[string]any{"current": float64(0)})
	if err != nil {
		t.Fatal(err)
	}
	for _, current := range []float64{85, 95} {
		if _, err := p.Eval(map[string]any{"current": current}); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Count(buf.String(), "Deprecated"); got != 1 {
		t.Errorf("got %d warnings, want 1", got)
	}
}

func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	orig := warnOut
	warnOut = buf
	warned = sync.Map{}
	t.Cleanup(func() {
		warnOut = orig
		warned = sync.Map{}
	})
	return buf
}
