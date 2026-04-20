package goxx

import (
	"io"

	"github.com/doors-dev/gox"
)

// Option configures a printer created by NewPrinter.
type Option interface {
	apply(p *printer)
}

// OptionFlat makes NewPrinter expand ordinary component jobs before they reach
// the base printer.
//
// Use it with OptionPrinter when your custom printer wants to handle the actual
// content stream and does not want to render or inspect gox.JobComp values
// itself. Parallel markers are still handled by NewPrinter.
func OptionFlat() Option {
	return optionFunc(func(p *printer) {
		p.flat = true
	})
}

// OptionWorkers sets the maximum number of parallel background worker tasks.
//
// The default is seven background workers, plus the caller goroutine, for eight
// render tasks in total. Passing zero skips the worker pool and starts plain
// goroutines for parallel tasks. Passing a negative value causes NewPrinter to
// panic.
func OptionWorkers(n int) Option {
	return optionFunc(func(p *printer) {
		p.workers = n
	})
}

// OptionPrinter lets you add your own printer extension to the rendering
// pipeline.
//
// NewPrinter calls f for each sequential chunk or parallel branch, passing the
// branch-local buffer that printer should write to. Use OptionFlat if your
// printer wants expanded content instead of gox.JobComp values.
//
// By default, NewPrinter uses gox.NewPrinter, which renders jobs directly to
// the provided io.Writer.
func OptionPrinter(f func(w io.Writer) gox.Printer) Option {
	return optionFunc(func(p *printer) {
		p.newPrinter = f
	})
}

type optionFunc func(p *printer)

func (f optionFunc) apply(p *printer) {
	f(p)
}
