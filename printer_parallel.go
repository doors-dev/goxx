package goxx

import (
	"context"
	"io"
	"log/slog"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx/internal/thread"
	"github.com/gammazero/deque"
)

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

func (p parallelPrinter) Send(j gox.Job) error {
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
		return p.Send(j)
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

func (p parallelPrinter) parallel(ctx context.Context, el gox.Elem) {
	branch := p.branch()
	p.thread.Go(func(t thread.Thread) error {
		p := parallelPrinter{
			newPrinter: p.newPrinter,
			queue:      branch,
			thread:     t,
		}
		p.initPrinter()
		cur := gox.NewCursor(ctx, p)
		return el(cur)
	})
}

func (p parallelPrinter) branch() *deque.Deque[any] {
	queue := new(deque.Deque[any])
	p.queue.PushBack(queue)
	p.initPrinter()
	return queue
}

func (p parallelPrinter) initPrinter() {
	buf := getBuffer()
	p.printer = p.newPrinter(buf)
	p.queue.PushBack(buf)
}

type CompParallel gox.Elem

func (p CompParallel) Main() gox.Elem {
	slog.Error("Parallel proxy is used outside goxx Printer")
	return gox.Elem(p)
}
