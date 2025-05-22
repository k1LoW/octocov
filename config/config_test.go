package config

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/text/language"
)

func TestMain(m *testing.M) {
	envCache := os.Environ()

	m.Run()

	if err := revertEnv(envCache); err != nil {
		_, _ = fmt.Fprint(os.Stderr, err) //nostyle:handlerrors
		os.Exit(1)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		wd      string
		path    string
		wantErr bool
	}{
		{rootTestdataDir(t), "", false},
		{filepath.Join(rootTestdataDir(t), "config"), "", false},
		{filepath.Join(rootTestdataDir(t), "config"), ".octocov.yml", false},
		{filepath.Join(rootTestdataDir(t), "config"), "no.yml", true},
	}
	for _, tt := range tests {
		c := New()
		c.wd = tt.wd
		if err := c.Load(tt.path); err != nil {
			if !tt.wantErr {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		} else {
			if tt.wantErr {
				t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
			}
		}
	}
}

func TestLoadComment(t *testing.T) {
	tests := []struct {
		path string
		want *Comment
	}{
		{"comment_enabled_octocov.yml", &Comment{}},
		{"comment_enabled_octocov2.yml", &Comment{If: "is_pull_request"}},
		{"comment_disabled_octocov.yml", nil},
	}
	for _, tt := range tests {
		c := New()
		p := filepath.Join(testdataDir(t), tt.path)
		if err := c.Load(p); err != nil {
			t.Fatal(err)
		}
		got := c.Comment
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Error(diff)
		}
	}
}

func TestLoadCentralPush(t *testing.T) {
	tests := []struct {
		path string
		want *Push
	}{
		{"central_push_enabled_octocov.yml", &Push{}},
		{"central_push_enabled_octocov2.yml", &Push{If: "is_default_branch"}},
		{"central_push_disabled_octocov.yml", nil},
	}
	for _, tt := range tests {
		c := New()
		p := filepath.Join(testdataDir(t), tt.path)
		if err := c.Load(p); err != nil {
			t.Fatal(err)
		}
		got := c.Central.Push
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Error(diff)
		}
	}
}

func TestLoadLocale(t *testing.T) {
	tests := []struct {
		path      string
		want      *language.Tag
		wantError bool
	}{
		{"locale_nothing.yml", nil, false},
		{"locale_empty.yml", nil, false},
		{"locale_ja.yml", &language.Japanese, false},
		{"locale_ja_uppercase.yml", &language.Japanese, false},
		{"locale_fr.yml", &language.French, false},
		{"locale_unkown.yml", nil, true},
	}
	for _, tt := range tests {
		c := New()
		t.Run(fmt.Sprintf("%v", tt.path), func(t *testing.T) {
			p := filepath.Join(testdataDir(t), tt.path)
			if err := c.Load(p); err != nil {
				if tt.wantError {
					return
				}
				t.Fatal(err)
			}
			got := c.Locale
			if tt.want == nil && got == nil {
				return
			}
			if diff := cmp.Diff(got.String(), tt.want.String(), nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestCoveragePaths(t *testing.T) {
	tests := []struct {
		paths      []string
		configPath string
		want       []string
	}{
		{[]string{"a/b/coverage.out"}, "path/to/.octocov.yml", []string{"path/to/a/b/coverage.out"}},
		{[]string{}, "path/to/.octocov.yml", []string{"path/to"}},
		{[]string{"a/b/coverage.out"}, ".octocov.yml", []string{"a/b/coverage.out"}},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.paths), func(t *testing.T) {
			c := New()
			c.path = filepath.FromSlash(tt.configPath)
			c.Coverage = &Coverage{
				Paths: tt.paths,
			}
			c.Build()
			got := c.Coverage.Paths
			var want []string
			for _, p := range tt.want {
				want = append(want, filepath.FromSlash(p))
			}
			if diff := cmp.Diff(got, want, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestCoverageAcceptable(t *testing.T) {
	// Pre-calculate special big.Rat values
	// For comparing 59.9999999999999 and 60
	almostSixty := new(big.Rat).SetFrac64(600000000000000-1, 10000000000000)

	// Value of 1/3
	oneThird := new(big.Rat).SetFrac64(1, 3)

	// Very small number
	verySmallNumber := new(big.Rat)
	_, _ = verySmallNumber.SetString("1/10000000000000000000")

	// 1/3 + small value
	oneThirdPlusSmall := new(big.Rat).Add(oneThird, verySmallNumber)

	tests := []struct {
		cond    string
		cov     float64
		prev    float64
		covRat  *big.Rat // To allow direct specification of big.Rat values
		prevRat *big.Rat // To allow direct specification of big.Rat values
		wantErr bool
		errMsg  string
	}{
		// Normal test cases
		{"60%", 50.0, 0, nil, nil, true, "code coverage is 50.0%. the condition in the `coverage.acceptable:` section is not met (`60%`)"},
		{"50%", 50.0, 0, nil, nil, false, ""},
		{"49.9%", 50.0, 0, nil, nil, false, ""},
		{"49.9", 50.0, 0, nil, nil, false, ""},
		{">= 60%", 50.0, 0, nil, nil, true, "code coverage is 50.0%. the condition in the `coverage.acceptable:` section is not met (`>= 60%`)"},
		{">= 50%", 50.0, 0, nil, nil, false, ""},
		{">= 49.9%", 50.0, 0, nil, nil, false, ""},
		{">= 49.9", 50.0, 0, nil, nil, false, ""},
		{">= 49.9%", 49.9, 0, nil, nil, false, ""},
		{">= 49.9", 49.9, 0, nil, nil, false, ""},
		{"> 49.9", 49.9, 0, nil, nil, true, "code coverage is 49.9%. the condition in the `coverage.acceptable:` section is not met (`> 49.9`)"},
		{">=60%", 50.0, 0, nil, nil, true, "code coverage is 50.0%. the condition in the `coverage.acceptable:` section is not met (`>=60%`)"},
		{">=50%", 50.0, 0, nil, nil, false, ""},
		{">=49.9%", 50.0, 0, nil, nil, false, ""},
		{">=49.9", 50.0, 0, nil, nil, false, ""},

		{"current >= 60%", 50.0, 0, nil, nil, true, "code coverage is 50.0%. the condition in the `coverage.acceptable:` section is not met (`current >= 60%`)"},
		{"current >= 60%", 59.9, 0, nil, nil, true, "code coverage is 59.9%. the condition in the `coverage.acceptable:` section is not met (`current >= 60%`)"},
		{"current >= 60%", 59.99, 0, nil, nil, true, "code coverage is 59.9%. the condition in the `coverage.acceptable:` section is not met (`current >= 60%`)"},
		{"current > prev", 50.0, 49.0, nil, nil, false, ""},
		{"diff >= 0", 50.0, 49.0, nil, nil, false, ""},
		{"current >= 50% && diff >= 0%", 50.0, 49.0, nil, nil, false, ""},

		// Test cases leveraging big.Rat precision
		// Test for small differences that cannot be represented by float64
		// Note: When big.Rat values are converted to float64, precision is lost, so wantErr is true
		{"current > 59.9999999999999", 0, 0, almostSixty, big.NewRat(0, 1), true, "code coverage is 59.9%. the condition in the `coverage.acceptable:` section is not met (`current > 59.9999999999999`)"}, // This test cannot be executed with float64
		{"current > prev", 0, 0, oneThirdPlusSmall, oneThird, true, "code coverage is 0.3%. the condition in the `coverage.acceptable:` section is not met (`current > prev`)"},                            // This test cannot be executed with float64
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			var covRat, prevRat *big.Rat

			if tt.covRat != nil && tt.prevRat != nil {
				// Use big.Rat values directly specified
				covRat = tt.covRat
				prevRat = tt.prevRat
			} else {
				// Create big.Rat from conventional float64 values
				covRat = big.NewRat(int64(tt.cov*10000), 10000)
				prevRat = big.NewRat(int64(tt.prev*10000), 10000)
			}

			if err := coverageAcceptable(covRat, prevRat, tt.cond); err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("got %v\nwant %v", err.Error(), tt.errMsg)
				}
			} else {
				if tt.wantErr {
					t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
				}
			}
		})
	}
}

func TestCodeToTestRatioAcceptable(t *testing.T) {
	// Pre-calculate special big.Rat values
	// Value of 1/3
	oneThird := new(big.Rat).SetFrac64(1, 3)

	// 1/3 + 1/3 + 1/3 (becomes 1.0 in float64, but is greater than 1.0 in big.Rat)
	threeThirds := new(big.Rat).Add(oneThird, oneThird)
	threeThirds = new(big.Rat).Add(threeThirds, oneThird)

	// Value of 1/7
	oneSeventh := new(big.Rat).SetFrac64(1, 7)

	// Very small number
	verySmallNumber := new(big.Rat)
	_, _ = verySmallNumber.SetString("1/10000000000000000000")

	// 1/7 + small value
	oneSeventhPlusSmall := new(big.Rat).Add(oneSeventh, verySmallNumber)

	tests := []struct {
		cond     string
		ratio    float64
		prev     float64
		ratioRat *big.Rat // To allow direct specification of big.Rat values
		prevRat  *big.Rat // To allow direct specification of big.Rat values
		wantErr  bool
	}{
		// Normal test cases
		{"1:1", 1.0, 0, nil, nil, false},
		{"1:1.1", 1.0, 0, nil, nil, true},
		{"1", 1.0, 0, nil, nil, false},
		{"1.1", 1.0, 0, nil, nil, true},
		{">= 1:1", 1.0, 0, nil, nil, false},
		{">= 1:1.1", 1.0, 0, nil, nil, true},
		{">= 1", 1.0, 0, nil, nil, false},
		{">= 1.1", 1.0, 0, nil, nil, true},
		{">=1:1", 1.0, 0, nil, nil, false},
		{">=1:1.1", 1.0, 0, nil, nil, true},
		{">=1", 1.0, 0, nil, nil, false},
		{">=1.1", 1.0, 0, nil, nil, true},

		{"current >= 1.1", 1.2, 1.1, nil, nil, false},
		{"current > prev", 1.2, 1.1, nil, nil, false},
		{"diff >= 0", 1.2, 1.1, nil, nil, false},
		{"current >= 1.1 && diff >= 0", 1.2, 1.1, nil, nil, false},

		// Test cases leveraging big.Rat precision
		// Note: When big.Rat values are converted to float64, precision is lost, so wantErr is true
		{"current > 1.0", 0, 0, threeThirds, big.NewRat(1, 1), true},    // Case like 1/3 + 1/3 + 1/3 > 1.0
		{"current > prev", 0, 0, oneSeventhPlusSmall, oneSeventh, true}, // Comparison of very small differences
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			var ratioRat, prevRat *big.Rat

			if tt.ratioRat != nil && tt.prevRat != nil {
				// Use big.Rat values directly specified
				ratioRat = tt.ratioRat
				prevRat = tt.prevRat
			} else {
				// Create big.Rat from conventional float64 values
				ratioRat = big.NewRat(int64(tt.ratio*10000), 10000)
				prevRat = big.NewRat(int64(tt.prev*10000), 10000)
			}

			if err := codeToTestRatioAcceptable(ratioRat, prevRat, tt.cond); err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
				}
			} else {
				if tt.wantErr {
					t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
				}
			}
		})
	}
}

func TestTestExecutionTimeAcceptable(t *testing.T) {
	// Pre-calculate special big.Rat values
	// Value of 1 minute
	oneMinute := new(big.Rat).SetInt64(int64(time.Minute))

	// Very small duration
	verySmallDuration := new(big.Rat).SetFrac64(1, 1000000000)

	// Value of 1 nanosecond
	oneNano := new(big.Rat).SetInt64(1)

	// Case of 59.999999999999 seconds (1 minute - small value)
	almostOneMinute := new(big.Rat).Sub(oneMinute, verySmallDuration)

	// 1 minute - 1 nanosecond
	oneMinuteMinusNano := new(big.Rat).Sub(oneMinute, oneNano)

	tests := []struct {
		cond    string
		ti      float64
		prev    float64
		tiRat   *big.Rat // To allow direct specification of big.Rat values
		prevRat *big.Rat // To allow direct specification of big.Rat values
		wantErr bool
	}{
		// Normal test cases
		{"1min", float64(time.Minute), 0, nil, nil, false},
		{"59s", float64(time.Minute), 0, nil, nil, true},
		{"61sec", float64(time.Minute), 0, nil, nil, false},
		{"<= 1min", float64(time.Minute), 0, nil, nil, false},
		{"<= 59s", float64(time.Minute), 0, nil, nil, true},
		{"<= 61sec", float64(time.Minute), 0, nil, nil, false},
		{"<=1min", float64(time.Minute), 0, nil, nil, false},
		{"<=59s", float64(time.Minute), 0, nil, nil, true},
		{"<=61sec", float64(time.Minute), 0, nil, nil, false},
		{"1 min", float64(time.Minute), 0, nil, nil, false},
		{"59 s", float64(time.Minute), 0, nil, nil, true},
		{"61 sec", float64(time.Minute), 0, nil, nil, false},

		{"1min1sec", float64(time.Minute), 0, nil, nil, false},
		{"<=1min1sec", float64(time.Minute), 0, nil, nil, false},
		{"<= 1 min 1 sec", float64(time.Minute), 0, nil, nil, false},
		{"current <= 1 min 1 sec", float64(time.Minute), 0, nil, nil, false},

		{"current <= 1min", float64(time.Minute), float64(59 * time.Second), nil, nil, false},
		{"current > prev", float64(time.Minute), float64(59 * time.Second), nil, nil, false},
		{"diff <= 1sec", float64(time.Minute), float64(59 * time.Second), nil, nil, false},
		{"current <= 1min && diff <= 1sec", float64(time.Minute), float64(59 * time.Second), nil, nil, false},

		// Test cases leveraging big.Rat precision
		// For test execution time, since the condition is "less than or equal to", it still meets the condition even when converted to float64
		{"current <= 1min", 0, 0, almostOneMinute, big.NewRat(0, 1), false}, // Case like 59.999999999999 seconds
		// Comparison of very small differences
		// Comparing 1 minute - 1 nanosecond < 1 minute, but when converted to float64, both become the same value
		{"current < prev", 0, 0, oneMinuteMinusNano, oneMinute, false}, // Comparison of very small differences
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			var tiRat, prevRat *big.Rat

			if tt.tiRat != nil && tt.prevRat != nil {
				// Use big.Rat values directly specified
				tiRat = tt.tiRat
				prevRat = tt.prevRat
			} else {
				// Create big.Rat from conventional float64 values
				tiRat = big.NewRat(int64(tt.ti*10000), 10000)
				prevRat = big.NewRat(int64(tt.prev*10000), 10000)
			}

			if err := testExecutionTimeAcceptable(tiRat, prevRat, tt.cond); err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
				}
			} else {
				if tt.wantErr {
					t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
				}
			}
		})
	}
}

func revertEnv(envCache []string) error {
	if err := clearEnv(); err != nil {
		return err
	}
	for _, e := range envCache {
		splitted := strings.Split(e, "=")
		if err := os.Setenv(splitted[0], splitted[1]); err != nil {
			return err
		}
	}
	return nil
}

func clearEnv() error {
	for _, e := range os.Environ() {
		splitted := strings.Split(e, "=")
		if err := os.Unsetenv(splitted[0]); err != nil {
			return err
		}
	}
	return nil
}

func rootTestdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(wd), "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
