package goxx

import (
	"context"
	"io"
	"log/slog"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx/internal/thread"
	"github.com/gammazero/deque"
)

// Parallel returns a proxy that schedules elem as a parallel subtree when it is
// rendered by NewPrinter.
//
// Use it for independent fragments that may wait on database queries, external
// API calls, or other slow work.
//
// Output order stays the same as the template order. When used with another
// printer, the subtree renders sequentially and logs a misuse warning.
func Parallel() gox.Proxy {
	return gox.ProxyFunc(func(cur gox.Cursor, el gox.Elem) error {
		if el == nil {
			return nil
		}
		return cur.Comp(CompParallel(el))
	})
}

type parallelPrinter struct {
	newPrinter func(w io.Writer) gox.Printer
	queue      *deque.Deque[any]
	thread     thread.Thread
	printer    gox.Printer
	flat       bool
}

func (p *parallelPrinter) Send(j gox.Job) error {
	compJob, ok := j.(*gox.JobComp)
	if !ok {
		return p.printer.Send(j)
	}
	if pc, ok := compJob.Comp.(CompParallel); ok {
		ctx := compJob.Ctx
		el := gox.Elem(pc)
		gox.Release(compJob)
		p.parallel(ctx, el)
		return nil
	}
	if !p.flat {
		return p.printer.Send(j)
	}
	ctx := compJob.Ctx
	el := compJob.Comp.Main()
	gox.Release(compJob)
	if el == nil {
		return nil
	}
	cur := gox.NewCursor(ctx, p)
	return el(cur)
}

func (p *parallelPrinter) parallel(ctx context.Context, el gox.Elem) {
	branch := p.branch()
	p.thread.Go(func(t thread.Thread) error {
		p := &parallelPrinter{
			newPrinter: p.newPrinter,
			queue:      branch,
			thread:     t,
			flat:       p.flat,
		}
		p.initPrinter()
		cur := gox.NewCursor(ctx, p)
		return el(cur)
	})
}

func (p *parallelPrinter) branch() *deque.Deque[any] {
	queue := new(deque.Deque[any])
	p.queue.PushBack(queue)
	p.initPrinter()
	return queue
}

func (p *parallelPrinter) initPrinter() {
	buf := getBuffer()
	p.printer = p.newPrinter(buf)
	p.queue.PushBack(buf)
}

// CompParallel marks an Elem for parallel rendering by NewPrinter.
//
// Prefer Parallel in templates and component code. CompParallel is exported so
// printer and proxy integrations can preserve the marker when wrapping or
// forwarding components.
type CompParallel gox.Elem

func (p CompParallel) Main() gox.Elem {
	slog.Warn(
		"goxx.Parallel used without goxx.NewPrinter; rendering subtree sequentially",
		"hint", "render with goxx.NewPrinter to enable parallel rendering",
	)
	return gox.Elem(p)
}
