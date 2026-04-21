package goxx

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx/internal/thread"
	"github.com/gammazero/deque"
)

// NewPrinter returns a GoX printer that renders gox.Comp and gox.Elem values
// with support for Parallel subtrees.
//
// To run part of a template in parallel, proxy that fragment to
// ~>(goxx.Parallel()). NewPrinter schedules those marked fragments on its
// worker pool while the rest of the template continues rendering.
//
// Output is buffered per parallel branch and drained to w in source order. By
// default the printer uses seven workers and gox.NewPrinter for sequential
// chunks. Use WithWorkers to tune or remove the worker limit. A negative
// worker count panics.
//
// NewPrinter is useful when you want to pass a printer to gox.Elem.Print. In
// HTTP handlers, prefer Render: it returns buffered output first, so you can set
// headers and a custom success status after rendering succeeds and before the
// response body is written.
//
// Use WriterError to check whether elem.Print failed because of the final
// io.Writer. Other render errors are returned before buffered output is written
// to w.
func NewPrinter(w io.Writer, opts ...Option) gox.Printer {
	p := newPrinter(opts)
	p.w = w
	return p
}

// Render renders comp into buffers and returns the buffered output.
//
// Render is the recommended entry point for HTTP handlers. It gives the same
// parallel rendering behavior as NewPrinter: fragments marked with
// ~>(goxx.Parallel()) are scheduled on the worker pool, and their buffered
// output is kept in template order. Rendering finishes before anything is
// written to the http.ResponseWriter. If rendering succeeds, set headers or a
// custom success status and then call WriteTo. If rendering fails, no output is
// returned, so the handler can still send an error response.
//
// Check context.Canceled and context.DeadlineExceeded separately: they mean the
// render context ended before rendering finished. If Render succeeds but
// WriteTo fails, that error came from the final writer.
func Render(ctx context.Context, comp gox.Comp, opts ...Option) (WriterToCloser, error) {
	printer := newPrinter(opts)
	out, err := printer.render(ctx, comp)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func newPrinter(opts []Option) printer {
	p := printer{}
	o := newOptions(opts)
	o.apply(&p)
	return p
}

type printer struct {
	w          io.Writer
	workers    int
	flat       bool
	newPrinter func(w io.Writer) gox.Printer
}

func (p printer) Send(j gox.Job) error {
	if j, ok := j.(*gox.JobComp); ok {
		comp := j.Comp
		ctx := j.Ctx
		gox.Release(j)
		return p.sendComp(ctx, comp)
	}
	slog.Warn(
		"goxx.NewPrinter received a non-component job; parallel rendering is disabled for this job",
		"job_type", fmt.Sprintf("%T", j),
	)
	printer := p.newPrinter(p.w)
	return printer.Send(j)
}

func (p printer) sendComp(ctx context.Context, comp gox.Comp) error {
	bt, err := p.render(ctx, comp)
	if err != nil {
		return err
	}
	_, err = bt.WriteTo(p.w)
	if err != nil {
		return WriteErr{err}
	}
	return nil
}

func (p printer) render(ctx context.Context, comp gox.Comp) (*bufferTree, error) {
	root := new(deque.Deque[any])
	err := thread.Root(ctx, p.workers, func(ctx context.Context, t thread.Thread) error {
		el := comp.Main()
		if el == nil {
			return nil
		}
		printer := &parallelPrinter{
			queue:      root,
			thread:     t,
			newPrinter: p.newPrinter,
			flat:       p.flat,
		}
		printer.initPrinter()
		cur := gox.NewCursor(ctx, printer)
		return el(cur)
	})
	bt := newBufferTree(root)
	if err != nil {
		bt.Close()
		return nil, err
	}
	return &bt, nil
}
