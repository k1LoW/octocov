package config

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
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
	// key is the config section this badge is written in, which is what an error about it has
	// to name. A run generates a badge per metric, and a central one a badge per metric per
	// repository, so every one of them carries a `colors:` an error could be about.
	key string
}

// BadgeColor is a message color of a badge and the condition it is picked by.
type BadgeColor struct {
	If    string `yaml:"if,omitempty"`
	Color string `yaml:"color"`
}

// validate validates the badge configurations set for central mode. Each is checked through
// the normalizer of the metric it belongs to, since that is what a condition is written in.
func (b *CentralBadges) validate() error {
	for _, v := range []struct {
		badge     *Badge
		normalize func(string) (string, error)
	}{
		{b.Coverage, normalizeCoverageCond},
		{b.CodeToTestRatio, normalizeCodeToTestRatioCond},
		{b.TestExecutionTime, normalizeTestExecutionTimeCond},
	} {
		if err := v.badge.Validate(v.normalize); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates the whole badge configuration, the conditions and the icon included,
// without a measured value. It is the up-front check of a central mode run, where a badge that
// cannot be rendered is worth hearing about before every datastore has been walked rather than
// after. normalize reads a condition the way the metric this badge belongs to writes one.
func (b *Badge) Validate(normalize func(string) (string, error)) error {
	if err := b.validateColors(); err != nil {
		return err
	}
	if _, err := b.compileColors(normalize); err != nil {
		return err
	}
	if _, err := b.icon(); err != nil {
		return err
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

// compileColors normalizes and compiles the condition of every entry of `colors:`, returning
// the programs in the order the entries are walked in. Entries without `if:` get a nil program,
// which the walk never runs.
func (b *Badge) compileColors(normalize func(string) (string, error)) ([]*vm.Program, error) {
	if b == nil {
		return nil, nil
	}
	programs := make([]*vm.Program, len(b.Colors))
	for i, c := range b.Colors {
		if c.If == "" {
			continue
		}
		cond, err := normalize(c.If)
		if err != nil {
			return nil, fmt.Errorf("%s.colors[%d].if: %w", b.section(), i, err)
		}
		p, err := expr.Compile(fmt.Sprintf("(%s) == true", cond))
		if err != nil {
			return nil, fmt.Errorf("%s.colors[%d].if: %w", b.section(), i, err)
		}
		programs[i] = p
	}
	return programs, nil
}

// validateColors validates the colors alone. It is what the render path checks, since the icon
// is read there anyway and reading it twice per badge would be the cost of checking it here.
func (b *Badge) validateColors() error {
	if b == nil {
		return nil
	}
	if b.LabelColor != "" {
		if _, err := badge.ParseColor(b.LabelColor); err != nil {
			return fmt.Errorf("%s.labelColor: %w", b.section(), err)
		}
	}
	for i, c := range b.Colors {
		if c.Color == "" {
			return fmt.Errorf("%s.colors[%d].color: is not set", b.section(), i)
		}
		if _, err := badge.ParseColor(c.Color); err != nil {
			return fmt.Errorf("%s.colors[%d].color: %w", b.section(), i, err)
		}
	}
	return nil
}

// build records where the badge is configured, which decides both how `icon:` resolves and how
// an error about the badge names itself.
func (b *Badge) build(key, dir string) {
	if b == nil {
		return
	}
	b.key = key
	b.dir = dir
}

// section returns the config section to name in an error. A badge that never went through
// Build, as in a test, falls back to the name of the key itself.
func (b *Badge) section() string {
	if b == nil || b.key == "" {
		return "badge"
	}
	return b.key
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
	// Every entry is compiled before any of them is evaluated. The walk stops at the first met
	// condition, so an entry the measured value happens to skip today would otherwise carry its
	// mistake until the day the value reaches it, and fail the run that gets there first.
	if err := b.validateColors(); err != nil {
		return "", err
	}
	programs, err := b.compileColors(normalize)
	if err != nil {
		return "", err
	}
	for i, c := range b.Colors {
		if c.If == "" {
			return badge.ParseColor(c.Color)
		}
		v, err := expr.Run(programs[i], map[string]any{"current": current})
		if err != nil {
			return "", fmt.Errorf("%s.colors[%d].if: %w", b.section(), i, err)
		}
		tf, ok := v.(bool)
		if !ok {
			return "", fmt.Errorf("%s.colors[%d].if: invalid condition `%s`", b.section(), i, c.If)
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
		return decodeDataURI(b.section(), b.Icon)
	}
	p := filepath.FromSlash(b.Icon)
	if !filepath.IsAbs(p) {
		p = filepath.Join(b.dir, p)
	}
	icon, err := os.ReadFile(filepath.Clean(p))
	if err != nil {
		return nil, fmt.Errorf("%s.icon: %w", b.section(), err)
	}
	return icon, nil
}

// decodeDataURI decodes a data URI of an icon. Only base64 encoded data is accepted because an
// icon is image data, which percent encoding would only make larger.
func decodeDataURI(section, s string) ([]byte, error) {
	meta, data, ok := strings.Cut(strings.TrimPrefix(s, "data:"), ",")
	if !ok || !strings.HasSuffix(meta, ";base64") {
		return nil, fmt.Errorf("%s.icon: only a base64 encoded data URI is supported", section)
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("%s.icon: %w", section, err)
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
