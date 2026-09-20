package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/k1LoW/octocov/badge"
	"github.com/k1LoW/octocov/internal"
)

// noIcon is the value of `icon:` that renders the badge without any icon.
const noIcon = "none"

// Badge is the configuration of a report badge.
type Badge struct {
	Path       string       `yaml:"path,omitempty"`
	Label      string       `yaml:"label,omitempty"`
	LabelColor string       `yaml:"labelColor,omitempty"`
	Icon       string       `yaml:"icon,omitempty"`
	Colors     []BadgeColor `yaml:"colors,omitempty"`
	// dir is the directory `icon:` is resolved from, which is the one holding the config file.
	dir string
}

// BadgeColor is a message color of a badge and the condition it is picked by.
type BadgeColor struct {
	If    string `yaml:"if,omitempty"`
	Color string `yaml:"color"`
}

// badges returns the badge configurations of the metrics measured in the repository.
func (c *Config) badges() []*Badge {
	var badges []*Badge
	if c.Coverage != nil {
		badges = append(badges, &c.Coverage.Badge)
	}
	if c.CodeToTestRatio != nil {
		badges = append(badges, &c.CodeToTestRatio.Badge)
	}
	if c.TestExecutionTime != nil {
		badges = append(badges, &c.TestExecutionTime.Badge)
	}
	return badges
}

// badges returns the badge configurations set for central mode.
func (b *CentralBadges) badges() []*Badge {
	var badges []*Badge
	for _, bb := range []*Badge{b.Coverage, b.CodeToTestRatio, b.TestExecutionTime} {
		if bb != nil {
			badges = append(badges, bb)
		}
	}
	return badges
}

// Validate validates the parts of the badge configuration that do not depend on a measured value.
func (b *Badge) Validate() error {
	if b == nil {
		return nil
	}
	if b.LabelColor != "" {
		if _, err := badge.ParseColor(b.LabelColor); err != nil {
			return fmt.Errorf("badge.labelColor: %w", err)
		}
	}
	for i, c := range b.Colors {
		if c.Color == "" {
			return fmt.Errorf("badge.colors[%d].color: is not set", i)
		}
		if _, err := badge.ParseColor(c.Color); err != nil {
			return fmt.Errorf("badge.colors[%d].color: %w", i, err)
		}
	}
	return nil
}

// RenderCoverage renders the badge of a measured coverage to w.
func (b *Badge) RenderCoverage(w io.Writer, cover float64) error {
	mc, err := b.CoverageColor(cover)
	if err != nil {
		return err
	}
	return b.render(w, "coverage", fmt.Sprintf("%.1f%%", floor1(cover)), mc)
}

// RenderCodeToTestRatio renders the badge of a measured code to test ratio to w.
func (b *Badge) RenderCodeToTestRatio(w io.Writer, ratio float64) error {
	mc, err := b.CodeToTestRatioColor(ratio)
	if err != nil {
		return err
	}
	return b.render(w, "code to test ratio", fmt.Sprintf("1:%.1f", floor1(ratio)), mc)
}

// RenderTestExecutionTime renders the badge of a measured test execution time, in nanoseconds, to w.
func (b *Badge) RenderTestExecutionTime(w io.Writer, nano float64) error {
	d := time.Duration(nano)
	mc, err := b.TestExecutionTimeColor(d)
	if err != nil {
		return err
	}
	return b.render(w, "test execution time", d.String(), mc)
}

// CoverageColor returns the message color of the coverage badge.
func (b *Badge) CoverageColor(cover float64) (string, error) {
	return b.color(cover, normalizeCoverageCond, defaultCoverageColor(cover))
}

// CodeToTestRatioColor returns the message color of the code to test ratio badge.
func (b *Badge) CodeToTestRatioColor(ratio float64) (string, error) {
	return b.color(ratio, normalizeCodeToTestRatioCond, defaultCodeToTestRatioColor(ratio))
}

// TestExecutionTimeColor returns the message color of the test execution time badge.
func (b *Badge) TestExecutionTimeColor(d time.Duration) (string, error) {
	return b.color(float64(d), normalizeTestExecutionTimeCond, defaultTestExecutionTimeColor(d))
}

// render renders the badge to w. label, message and messageColor are what the measured metric
// says, and `label:`, `labelColor:` and `icon:` of the configuration override the appearance.
func (b *Badge) render(w io.Writer, label, message, messageColor string) error {
	if b != nil && b.Label != "" {
		label = b.Label
	}
	bb := badge.New(label, message)
	if messageColor != "" {
		if err := bb.SetMessageColor(messageColor); err != nil {
			return err
		}
	}
	if b != nil && b.LabelColor != "" {
		if err := bb.SetLabelColor(b.LabelColor); err != nil {
			return err
		}
	}
	icon, err := b.icon()
	if err != nil {
		return err
	}
	if len(icon) > 0 {
		if err := bb.AddIcon(icon); err != nil {
			return err
		}
	}
	return bb.Render(w)
}

// color returns the color of the first condition of `colors:` that current meets. def, the
// color of the built-in thresholds, is returned when `colors:` is not set or nothing matches,
// so that a configuration listing only the colors it cares about keeps the rest as they were.
func (b *Badge) color(current float64, normalize func(string) (string, error), def string) (string, error) {
	if b == nil {
		return def, nil
	}
	for i, c := range b.Colors {
		if c.Color == "" {
			return "", fmt.Errorf("badge.colors[%d].color: is not set", i)
		}
		if c.If == "" {
			return badge.ParseColor(c.Color)
		}
		cond, err := normalize(c.If)
		if err != nil {
			return "", err
		}
		v, err := expr.Eval(fmt.Sprintf("(%s) == true", cond), map[string]any{"current": current})
		if err != nil {
			return "", err
		}
		tf, ok := v.(bool)
		if !ok {
			return "", fmt.Errorf("invalid condition `%s`", c.If)
		}
		if tf {
			return badge.ParseColor(c.Color)
		}
	}
	return def, nil
}

// icon returns the icon image of the badge. The embedded octocov icon is used unless `icon:`
// names another one, and `none` leaves the badge without an icon.
func (b *Badge) icon() ([]byte, error) {
	if b == nil || b.Icon == "" {
		return internal.Icon, nil
	}
	if b.Icon == noIcon {
		return nil, nil
	}
	if strings.HasPrefix(b.Icon, "data:") {
		return decodeDataURI(b.Icon)
	}
	p := filepath.FromSlash(b.Icon)
	if !filepath.IsAbs(p) {
		p = filepath.Join(b.dir, p)
	}
	return os.ReadFile(filepath.Clean(p))
}

// decodeDataURI decodes a data URI of an icon. Only base64 encoded data is accepted because an
// icon is image data, which percent encoding would only make larger.
func decodeDataURI(s string) ([]byte, error) {
	meta, data, ok := strings.Cut(strings.TrimPrefix(s, "data:"), ",")
	if !ok || !strings.HasSuffix(meta, ";base64") {
		return nil, errors.New("badge.icon: only a base64 encoded data URI is supported")
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("badge.icon: %w", err)
	}
	return b, nil
}

func defaultCoverageColor(cover float64) string {
	switch {
	case cover >= 80.0:
		return badge.ColorGreen
	case cover >= 60.0:
		return badge.ColorYellowGreen
	case cover >= 40.0:
		return badge.ColorYellow
	case cover >= 20.0:
		return badge.ColorOrange
	default:
		return badge.ColorRed
	}
}

func defaultCodeToTestRatioColor(ratio float64) string {
	switch {
	case ratio >= 1.2:
		return badge.ColorGreen
	case ratio >= 1.0:
		return badge.ColorYellowGreen
	case ratio >= 0.8:
		return badge.ColorYellow
	case ratio >= 0.6:
		return badge.ColorOrange
	default:
		return badge.ColorRed
	}
}

func defaultTestExecutionTimeColor(d time.Duration) string {
	switch {
	case d < 5*time.Minute:
		return badge.ColorGreen
	case d < 10*time.Minute:
		return badge.ColorYellowGreen
	case d < 15*time.Minute:
		return badge.ColorYellow
	case d < 20*time.Minute:
		return badge.ColorOrange
	default:
		return badge.ColorRed
	}
}
