package goxx

import (
	"bytes"
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
// chunks. Use OptionWorkers to tune or remove the worker limit. A negative
// worker count panics.
//
// Use WriterError to check whether elem.Print failed because of the final
// io.Writer. Other render errors are returned before buffered output is written
// to w. In HTTP handlers, that usually leaves room to send an error response
// with the correct status code. Check for context.Canceled and
// context.DeadlineExceeded separately: they mean the render context ended
// before rendering finished.
func NewPrinter(w io.Writer, opts ...Option) gox.Printer {
	p := printer{
		w:          w,
		workers:    7,
		newPrinter: gox.NewPrinter,
	}
	for _, opt := range opts {
		opt.apply(&p)
	}
	if p.workers < 0 {
		panic("Can't have negative worker count")
	}
	return p
}

type printer struct {
	w          io.Writer
	workers    int
	newPrinter func(w io.Writer) gox.Printer
	flat       bool
}

func (p printer) Send(j gox.Job) error {
	if j, ok := j.(*gox.JobComp); ok {
		comp := j.Comp
		ctx := j.Ctx
		gox.Release(j)
		return p.printComp(ctx, comp)
	}
	slog.Warn(
		"goxx.NewPrinter received a non-component job; parallel rendering is disabled for this job",
		"job_type", fmt.Sprintf("%T", j),
	)
	printer := p.newPrinter(p.w)
	return printer.Send(j)
}

func (p printer) printComp(ctx context.Context, comp gox.Comp) error {
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
	return p.drain(root, err)
}

func (p printer) drain(root *deque.Deque[any], err error) error {
	stack := []*deque.Deque[any]{root}
main:
	queue := stack[len(stack)-1]
	for item := range queue.IterPopFront() {
		switch item := item.(type) {
		case *bytes.Buffer:
			if err != nil {
				putBuffer(item)
				continue
			}
			_, writeErr := item.WriteTo(p.w)
			putBuffer(item)
			if writeErr != nil {
				err = WriteErr{writeErr}
			}
		case *deque.Deque[any]:
			stack = append(stack, item)
			goto main
		default:
			panic("Unexpected item type")
		}
	}
	stack[len(stack)-1] = nil
	stack = stack[:len(stack)-1]
	if len(stack) != 0 {
		goto main
	}
	return err
}
