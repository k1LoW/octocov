package report

import "golang.org/x/text/language"

type ReportOptions struct {
	Locale *language.Tag
}

type ReportOption func(*ReportOptions)

func Locale(locale *language.Tag) ReportOption {
	return func(args *ReportOptions) {
		args.Locale = locale
	}
}
