package config

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/badge"
)

func TestBadgeCoverageColor(t *testing.T) {
	tests := []struct {
		name    string
		badge   *Badge
		cover   float64
		want    string
		wantErr bool
	}{
		{"colors is not set", &Badge{}, 85.0, badge.ColorGreen, false},
		{"badge is not set", nil, 10.0, badge.ColorRed, false},
		{"the first met condition is used", &Badge{Colors: []BadgeColor{
			{If: "90%", Color: "blue"},
			{If: "50%", Color: "orange"},
		}}, 85.0, badge.ColorOrange, false},
		{"condition can be an expression", &Badge{Colors: []BadgeColor{
			{If: "current >= 80 && current < 90", Color: "#123456"},
		}}, 85.0, "#123456", false},
		{"condition can be a comparison only", &Badge{Colors: []BadgeColor{
			{If: "> 80%", Color: "blue"},
		}}, 80.0, badge.ColorGreen, false},
		{"the entry without if is the fallback", &Badge{Colors: []BadgeColor{
			{If: "90%", Color: "blue"},
			{Color: "lightgrey"},
		}}, 85.0, badge.ColorLightGrey, false},
		{"no met condition falls back to the built-in thresholds", &Badge{Colors: []BadgeColor{
			{If: "90%", Color: "blue"},
		}}, 45.0, badge.ColorYellow, false},
		{"invalid color", &Badge{Colors: []BadgeColor{
			{If: "10%", Color: "octocov"},
		}}, 85.0, "", true},
		{"invalid color of an entry the value does not reach", &Badge{Colors: []BadgeColor{
			{If: "80%", Color: "green"},
			{If: "10%", Color: "octocov"},
		}}, 85.0, "", true},
		{"invalid condition", &Badge{Colors: []BadgeColor{
			{If: "current >>= 10", Color: "blue"},
		}}, 85.0, "", true},
		{"invalid condition of an entry the value does not reach", &Badge{Colors: []BadgeColor{
			{If: "80%", Color: "green"},
			{If: "current >>= 10", Color: "blue"},
		}}, 85.0, "", true},
		{"color is not set", &Badge{Colors: []BadgeColor{
			{If: "10%"},
		}}, 85.0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.badge.CoverageColor(tt.cover)
			if err != nil {
				if !tt.wantErr {
					t.Fatal(err)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("want error")
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestBadgeCodeToTestRatioColor(t *testing.T) {
	tests := []struct {
		name  string
		badge *Badge
		ratio float64
		want  string
	}{
		{"colors is not set", &Badge{}, 1.3, badge.ColorGreen},
		{"threshold can be written as a ratio", &Badge{Colors: []BadgeColor{
			{If: "1:1.5", Color: "green"},
			{If: "1:0.5", Color: "yellow"},
		}}, 1.0, badge.ColorYellow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.badge.CodeToTestRatioColor(tt.ratio)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestBadgeTestExecutionTimeColor(t *testing.T) {
	tests := []struct {
		name  string
		badge *Badge
		d     time.Duration
		want  string
	}{
		{"colors is not set", &Badge{}, 3 * time.Minute, badge.ColorGreen},
		{"threshold can be written as a duration", &Badge{Colors: []BadgeColor{
			{If: "30sec", Color: "green"},
			{If: "5min", Color: "orange"},
		}}, 3 * time.Minute, badge.ColorOrange},
		{"condition can be a comparison only", &Badge{Colors: []BadgeColor{
			{If: "> 10min", Color: "red"},
		}}, 20 * time.Minute, badge.ColorRed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.badge.TestExecutionTimeColor(tt.d)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestBadgeRenderCoverage(t *testing.T) {
	icon, err := os.ReadFile(filepath.Join(testdataDir(t), "icon.svg"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		badge      *Badge
		want       []string
		wantNot    []string
		wantErrStr string
	}{
		{"badge is not set", nil, []string{">coverage<", ">51.2%<", "<image"}, nil, ""},
		{"default", &Badge{}, []string{">coverage<", ">51.2%<", badge.ColorYellow, "<image"}, nil, ""},
		{"label", &Badge{Label: "cov"}, []string{">cov<"}, []string{">coverage<"}, ""},
		{"labelColor", &Badge{LabelColor: "blue"}, []string{badge.ColorBlue}, nil, ""},
		{"colors", &Badge{Colors: []BadgeColor{{If: "50%", Color: "#123456"}}}, []string{"#123456"}, []string{badge.ColorYellow}, ""},
		{"icon none", &Badge{Icon: "none"}, nil, []string{"<image"}, ""},
		{"icon file", &Badge{Icon: "icon.svg", dir: testdataDir(t)}, []string{"<image"}, nil, ""},
		{"icon data URI", &Badge{Icon: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(icon)}, []string{"<image"}, nil, ""},
		{"icon data URI that is not base64", &Badge{Icon: "data:image/svg+xml,<svg/>"}, nil, nil, "only a base64 encoded data URI is supported"},
		{"icon file that does not exist", &Badge{Icon: "no_such_icon.svg", dir: testdataDir(t), key: "coverage.badge"}, nil, nil, "coverage.badge.icon: open"},
		{"icon file that is not an image", &Badge{Icon: "badge_octocov.yml", dir: testdataDir(t), key: "coverage.badge"}, nil, nil, "coverage.badge.icon: invalid icon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(bytes.Buffer)
			if err := tt.badge.RenderCoverage(got, 51.23); err != nil {
				if tt.wantErrStr == "" {
					t.Fatal(err)
				}
				if !strings.Contains(err.Error(), tt.wantErrStr) {
					t.Errorf("got %v\nwant to contain %v", err, tt.wantErrStr)
				}
				return
			}
			if tt.wantErrStr != "" {
				t.Fatal("want error")
			}
			for _, w := range tt.want {
				if !strings.Contains(got.String(), w) {
					t.Errorf("want to contain %v", w)
				}
			}
			for _, w := range tt.wantNot {
				if strings.Contains(got.String(), w) {
					t.Errorf("want not to contain %v", w)
				}
			}
		})
	}
}

func TestBadgeLoad(t *testing.T) {
	c := New()
	if err := c.Load(filepath.Join(testdataDir(t), "badge_octocov.yml")); err != nil {
		t.Fatal(err)
	}
	c.Build()

	got := c.Coverage.Badge
	want := Badge{
		Path:       "docs/coverage.svg",
		Label:      "cov",
		LabelColor: "#24292E",
		Icon:       "icon.svg",
		Colors: []BadgeColor{
			{If: "current >= 80%", Color: "green"},
			{Color: "red"},
		},
		dir: testdataDir(t),
		key: "coverage.badge",
	}
	if diff := cmp.Diff(got, want, cmp.AllowUnexported(Badge{})); diff != "" {
		t.Error(diff)
	}

	gotc := c.Central.Badges.Coverage
	wantc := &Badge{
		Icon:   "none",
		Colors: []BadgeColor{{If: "80%", Color: "green"}},
		dir:    testdataDir(t),
		key:    "central.badges.coverage",
	}
	if diff := cmp.Diff(gotc, wantc, cmp.AllowUnexported(Badge{})); diff != "" {
		t.Error(diff)
	}
}

func TestBadgeValidate(t *testing.T) {
	tests := []struct {
		name    string
		badge   *Badge
		wantErr bool
	}{
		{"badge is not set", nil, false},
		{"empty", &Badge{}, false},
		{"valid", &Badge{LabelColor: "#24292E", Colors: []BadgeColor{{If: "80%", Color: "green"}, {Color: "red"}}}, false},
		{"invalid labelColor", &Badge{LabelColor: "octocov"}, true},
		{"invalid color", &Badge{Colors: []BadgeColor{{If: "80%", Color: "octocov"}}}, true},
		{"color is not set", &Badge{Colors: []BadgeColor{{If: "80%"}}}, true},
		{"icon file that does not exist", &Badge{Icon: "no_such_icon.svg"}, true},
		{"icon data URI that is not base64", &Badge{Icon: "data:image/svg+xml,<svg/>"}, true},
		{"invalid condition", &Badge{Colors: []BadgeColor{{If: "current >>= 10", Color: "green"}}}, true},
		{"icon file that is not an image", &Badge{Icon: "badge_octocov.yml", dir: testdataDir(t)}, true},
		{"invalid condition of an entry a value would not reach", &Badge{Colors: []BadgeColor{
			{If: "80%", Color: "green"},
			{If: "current >>= 10", Color: "red"},
		}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.badge.Validate(normalizeCoverageCond); err != nil != tt.wantErr {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		})
	}
}
