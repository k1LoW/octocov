package report

import (
	_ "embed"
	"fmt"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var locale *language.Tag

func SetLocale(l *language.Tag) {
	locale = l
}

func convertFormat(v interface{}) string {
	if locale != nil {
		p := message.NewPrinter(*locale)
		return p.Sprint(number.Decimal(v))
	}

	switch vv := v.(type) {
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", vv)
	case float64:
		if isInt(vv) {
			return fmt.Sprintf("%d", int(vv))
		}
		return fmt.Sprintf("%.1f", vv)
	default:
		panic(fmt.Errorf("convert format error .Unknown type:%v", vv))
	}
}
