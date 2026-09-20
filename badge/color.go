package badge

import (
	"fmt"
	"strings"
)

// Named colors of the shields.io palette.
// https://github.com/badges/shields/blob/7d452472defa0e0bd71d6443393e522e8457f856/badge-maker/lib/color.js#L8-L12
const (
	ColorBrightGreen = "#44CC11"
	ColorGreen       = "#97CA00"
	ColorYellowGreen = "#A4A61D"
	ColorYellow      = "#DFB317"
	ColorOrange      = "#FE7D37"
	ColorRed         = "#E05D44"
	ColorBlue        = "#007EC6"
	ColorGrey        = "#555555"
	ColorLightGrey   = "#9F9F9F"
)

var namedColors = map[string]string{
	"brightgreen": ColorBrightGreen,
	"green":       ColorGreen,
	"yellowgreen": ColorYellowGreen,
	"yellow":      ColorYellow,
	"orange":      ColorOrange,
	"red":         ColorRed,
	"blue":        ColorBlue,
	"grey":        ColorGrey,
	"gray":        ColorGrey,
	"lightgrey":   ColorLightGrey,
	"lightgray":   ColorLightGrey,
}

// ParseColor resolves a color name or a hex RGB notation to a hex RGB color.
func ParseColor(c string) (string, error) {
	v := strings.TrimSpace(c)
	if hex, ok := namedColors[strings.ToLower(v)]; ok {
		return hex, nil
	}
	rgb := strings.ToUpper(strings.TrimPrefix(v, "#"))
	if !rgbRe.MatchString(rgb) {
		return "", fmt.Errorf("invalid color: %s", c)
	}
	return fmt.Sprintf("#%s", rgb), nil
}
