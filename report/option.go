package report

import "golang.org/x/text/language"

type Options struct {
	Locale *language.Tag
}

type Option func(*Options)

func Locale(locale *language.Tag) Option {
	return func(args *Options) {
		args.Locale = locale
	}
}
