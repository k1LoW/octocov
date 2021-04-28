package badge

import (
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"text/template"

	"github.com/mattn/go-runewidth"
)

const defaultKeyColor = "#24292E"
const defaultValueColor = "#28A745"

type Badge struct {
	Key        string
	Value      string
	KeyColor   string
	ValueColor string
}

//go:embed badge.svg.tmpl
var badgeTmpl []byte

func New(k, v string) *Badge {
	return &Badge{
		Key:        k,
		Value:      v,
		KeyColor:   defaultKeyColor,
		ValueColor: defaultValueColor,
	}
}

func (b *Badge) Render(wr io.Writer) error {
	tmpl := template.Must(template.New("badge").Parse(string(badgeTmpl)))

	// https://github.com/badges/shields/tree/master/spec
	kw := 6 + stringWidth(b.Key) + 4
	vw := 4 + stringWidth(b.Value) + 6
	kx := kw * 10 / 2
	vx := (kw * 10) + (vw * 10 / 2)

	d := map[string]interface{}{
		"Key":        b.Key,
		"Value":      b.Value,
		"KeyColor":   b.KeyColor,
		"ValueColor": b.ValueColor,
		"Width":      kw + vw,
		"KeyWidth":   kw,
		"ValueWidth": vw,
		"KeyX":       kx,
		"ValueX":     vx,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}

	return nil
}

func stringWidth(s string) int {
	return runewidth.StringWidth(s) * 8 // TODO: 8 is heuristic
}

func ColorToHexRGB(c color.Color) string {
	rgba := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
