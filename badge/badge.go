package badge

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/antchfx/xmlquery"
	"github.com/golang/freetype/truetype"
	issvg "github.com/h2non/go-is-svg"
	"golang.org/x/exp/utf8string"
	"golang.org/x/image/font"
)

const defaultLabelColor = "#24292E"
const defaultMessageColor = "#007EC6"
const fontSize = 11
const dpi = 72

var rgbRe = regexp.MustCompile(`^[0-9A-F]{6}$`)

type Badge struct {
	Label        string
	Message      string
	LabelColor   string
	MessageColor string
	Icon         []byte
	drawer       *font.Drawer
}

//go:embed badge.svg.tmpl
var badgeTmpl []byte

// https://github.com/googlefonts/noto-fonts/blob/main/hinted/ttf/NotoSans/NotoSans-Medium.ttf
//go:embed NotoSans-Medium.ttf
var noto []byte

// New returns *Badge.
func New(l, m string) *Badge {
	ttf, err := truetype.Parse(noto)
	if err != nil {
		panic(err) //nostyle:dontpanic
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

// AddIcon add icon image to badge.
func (b *Badge) AddIcon(imgf []byte) error {
	b.Icon = imgf
	return nil
}

// AddIconFile add icon image file to badge.
func (b *Badge) AddIconFile(f string) error {
	imgf, err := os.ReadFile(filepath.Clean(f))
	if err != nil {
		return err
	}
	return b.AddIcon(imgf)
}

func (b *Badge) SetLabelColor(c any) error {
	rgb, err := castColor(c)
	if err != nil {
		return err
	}
	b.LabelColor = rgb
	return nil
}

func (b *Badge) SetMessageColor(c any) error {
	rgb, err := castColor(c)
	if err != nil {
		return err
	}
	b.MessageColor = rgb
	return nil
}

func castColor(c any) (string, error) {
	switch v := c.(type) {
	case string:
		rgb := strings.ToUpper(strings.TrimPrefix(v, "#"))
		if !rgbRe.MatchString(rgb) {
			return "", fmt.Errorf("invalid color: %s", v)
		}
		return fmt.Sprintf("#%s", rgb), nil
	default:
		return "", fmt.Errorf("invalid color: %v", v)
	}
}

// Render badge.
func (b *Badge) Render(wr io.Writer) error {
	tmpl := template.Must(template.New("badge").Parse(string(badgeTmpl)))

	// https://github.com/badges/shields/tree/master/spec
	lw := 6 + b.stringWidth(b.Label) + 4
	mw := 4 + b.stringWidth(b.Message) + 6
	lx := lw * 10 / 2
	mx := (lw * 10) + (mw * 10 / 2)
	iw := 0.0
	var icon string
	if len(b.Icon) != 0 {
		iw = 15.5
		if issvg.Is(b.Icon) {
			imgdoc, err := xmlquery.Parse(bytes.NewReader(b.Icon))
			if err != nil {
				return err
			}
			s := xmlquery.FindOne(imgdoc, "//svg")
			icon = fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString([]byte(s.OutputXML(true))))
		} else {
			_, format, err := image.DecodeConfig(bytes.NewReader(b.Icon))
			if err != nil {
				return err
			}
			icon = fmt.Sprintf("data:image/%s;base64,%s", format, base64.StdEncoding.EncodeToString(b.Icon))
		}
	}

	d := map[string]any{
		"Label":        b.Label,
		"Message":      b.Message,
		"LabelColor":   b.LabelColor,
		"MessageColor": b.MessageColor,
		"Width":        lw + mw + iw,
		"LabelWidth":   lw + iw,
		"MessageWidth": mw,
		"LabelX":       lx + (iw * 10),
		"MessageX":     mx + (iw * 10),
		"Icon":         icon,
	}
	if err := tmpl.Execute(wr, d); err != nil {
		return err
	}

	return nil
}

func (b *Badge) stringWidth(s string) float64 {
	var converted []rune
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

// ColorToHexRGB return Hex RGB from color.Color.
func ColorToHexRGB(c color.Color) string {
	rgba, ok := color.NRGBAModel.Convert(c).(color.NRGBA)
	if !ok {
		return "#000000"
	}
	return fmt.Sprintf("#%.2x%.2x%.2x", rgba.R, rgba.G, rgba.B)
}
