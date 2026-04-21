package goxx

import (
	"io"

	"github.com/doors-dev/gox"
)

func newOptions(opts []Option) options {
	o := options{
		workers:    7,
		newPrinter: gox.NewPrinter,
	}
	for _, opt := range opts {
		opt.apply(&o)
	}
	if o.workers < 0 {
		panic("Can't have negative worker count")
	}
	if o.newPrinter == nil {
		panic("new printer factory can't be nil")
	}
	return o
}

type options struct {
	workers    int
	flat       bool
	newPrinter func(w io.Writer) gox.Printer
}

func (o options) apply(p *printer) {
	p.workers = o.workers
	p.flat = o.flat
	p.newPrinter = o.newPrinter
}

// Option configures a printer created by NewPrinter.
type Option interface {
	apply(p *options)
}

// WithFlat makes NewPrinter expand ordinary component jobs before they reach
// the base printer.
//
// Use it with WithPrinter when your custom printer wants to handle the actual
// content stream and does not want to render or inspect gox.JobComp values
// itself. Parallel markers are still handled by NewPrinter.
func WithFlat() Option {
	return optionFunc(func(o *options) {
		o.flat = true
	})
}

// WithWorkers sets the maximum number of parallel background worker tasks.
//
// The default is seven background workers, plus the caller goroutine, for eight
// render tasks in total. Passing zero skips the worker pool and starts plain
// goroutines for parallel tasks. Passing a negative value causes NewPrinter to
// panic.
func WithWorkers(n int) Option {
	return optionFunc(func(o *options) {
		o.workers = n
	})
}

// WithPrinter lets you add your own printer extension to the rendering
// pipeline.
//
// NewPrinter calls f for each sequential chunk or parallel branch, passing the
// branch-local buffer that printer should write to. Use WithFlat if your
// printer wants expanded content instead of gox.JobComp values.
//
// By default, NewPrinter uses gox.NewPrinter, which renders jobs directly to
// the provided io.Writer.
func WithPrinter(f func(w io.Writer) gox.Printer) Option {
	return optionFunc(func(o *options) {
		o.newPrinter = f
	})
}

type optionFunc func(o *options)

func (f optionFunc) apply(o *options) {
	f(o)
}
