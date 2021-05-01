package cmd

const (
	// https://github.com/badges/shields/blob/7d452472defa0e0bd71d6443393e522e8457f856/badge-maker/lib/color.js#L8-L12
	green       = "#97CA00"
	yellowgreen = "#A4A61D"
	yellow      = "#DFB317"
	orange      = "#FE7D37"
	red         = "#E05D44"
)

func coverageColor(cover float64) string {
	switch {
	case cover >= 80.0:
		return green
	case cover >= 60.0:
		return yellowgreen
	case cover >= 40.0:
		return yellow
	case cover >= 20.0:
		return orange
	default:
		return red
	}
}
