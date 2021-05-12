package badge

import (
	_ "embed"
	"fmt"
	"image/color"
	"io"
	"text/template"

	"github.com/golang/freetype/truetype"
	"golang.org/x/exp/utf8string"
	"golang.org/x/image/font"
)

const defaultLabelColor = "#24292E"
const defaultMessageColor = "#007EC6"
const fontSize = 11
const dpi = 72

type Badge struct {
	Label        string
	Message      string
	LabelColor   string
	MessageColor string
	drawer       *font.Drawer
}

//go:embed badge.svg.tmpl
var badgeTmpl []byte

// https://github.com/googlefonts/noto-fonts/blob/main/hinted/ttf/NotoSans/NotoSans-Medium.ttf
//go:embed NotoSans-Medium.ttf
var noto []byte

func New(l, m string) *Badge {
	ttf, err := truetype.Parse(noto)
	if err != nil {
		panic(err)
	}

	return &Badge{
		Label:        l,
		Message:      m,
		LabelColor:   defaultLabelColor,
		MessageColor: defaultMessageColor,
		drawer: &font.Drawer{
			Face: truetype.NewFace(ttf, &truetype.Options{
				Size:    fontSize,
				DPI:     dpi,
				Hinting: font.HintingFull,
			}),
		},
	}
}

func (b *Badge) Render(wr io.Writer) error {
	tmpl := template.Must(template.New("badge").Parse(string(badgeTmpl)))

	// https://github.com/badges/shields/tree/master/spec
	lw := 6 + b.stringWidth(b.Label) + 4
	mw := 4 + b.stringWidth(b.Message) + 6
	lx := lw * 10 / 2
	mx := (lw * 10) + (mw * 10 / 2)

	d := map[string]interface{}{
		"Label":        b.Label,
		"Message":      b.Message,
		"LabelColor":   b.LabelColor,
		"MessageColor": b.MessageColor,
		"Width":        lw + mw,
		"LabelWidth":   lw,
		"MessageWidth": mw,
		"LabelX":       lx,
		"MessageX":     mx,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}

	return nil
}

func (b *Badge) stringWidth(s string) float64 {
	converted := []rune{}
	for _, c := range s {
		if utf8string.NewString(string([]rune{c})).IsASCII() {
			converted = append(converted, c)
		} else {
			converted = append(converted, '%') // because the width of the `%` character is wider
		}
	}
	w := b.drawer.MeasureString(string(converted))
	return float64(w)/64 + 10 // 10 is heuristic
}

func ColorToHexRGB(c color.Color) string {
	rgba := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
