package goxx

import (
	"io"

	"github.com/doors-dev/gox"
)

type Option interface {
	apply(p *printer)
}

func OptionFlat() Option {
	return optionFunc(func(p *printer) {
		p.flat = true
	})
}

func OptionWorkers(n int) Option {
	return optionFunc(func(p *printer) {
		p.workers = n
	})
}

func OptionPrinter(f func(w io.Writer) gox.Printer) Option {
	return optionFunc(func(p *printer) {
		p.newPrinter = f
	})
}

type optionFunc func(p *printer)

func (f optionFunc) apply(p *printer) {
	f(p)
}
