package goxx

import (
	"context"
	"errors"

	"github.com/doors-dev/gox"
)

// ProxyMod returns a proxy that applies mod to the first element rendered by
// elem.
//
// Use it to build helpers that attach attributes or attribute modifiers to
// another element or component. If elem starts with a component or container,
// ProxyMod carries mod forward until it reaches the first element. Text or other
// non-element output before that element is an error. The modifier is applied
// once; later sibling elements are left unchanged.
//
// Parallel markers are preserved, so the wrapped subtree can still be
// scheduled by NewPrinter.
func ProxyMod(mod gox.Modify) gox.Proxy {
	return gox.ProxyFunc(func(cur gox.Cursor, el gox.Elem) error {
		printer := &modPrinter{
			mod: mod,
			cur: cur,
		}
		cursor := gox.NewCursor(cur.Context(), printer)
		return el(cursor)
	})
}

type modPrinter struct {
	mod gox.Modify
	cur gox.Cursor
}

func (m *modPrinter) Send(j gox.Job) error {
	if m.mod == nil {
		return m.cur.Send(j)
	}
	comp, ok := j.(*gox.JobComp)
	if ok {
		mod := m.mod
		m.mod = nil
		return m.printComp(mod, comp)
	}
	open, ok := j.(*gox.JobHeadOpen)
	if ok {
		if open.Kind == gox.KindContainer {
			return m.cur.Send(open)
		}
		mod := m.mod
		m.mod = nil
		return m.printHead(mod, open)
	}
	m.mod = nil
	return errors.New("cannot attach an attribute modifier: unexpected job type")
}

func (m *modPrinter) printComp(mod gox.Modify, job *gox.JobComp) error {
	ctx := job.Ctx
	comp := job.Comp
	gox.Release(job)
	if pc, ok := comp.(CompParallel); ok {
		return m.submitCompParallel(mod, ctx, pc)
	}
	return m.submitComp(mod, ctx, comp)
}

func (m *modPrinter) submitCompParallel(mod gox.Modify, ctx context.Context, comp CompParallel) error {
	el := gox.Elem(comp)
	return m.cur.CompCtx(ctx, CompParallel(gox.Elem(func(cur gox.Cursor) error {
		p := &modPrinter{mod: mod, cur: cur}
		cur = gox.NewCursor(cur.Context(), p)
		return el(cur)
	})))
}

func (m *modPrinter) submitComp(mod gox.Modify, ctx context.Context, comp gox.Comp) error {
	return m.cur.CompCtx(ctx, gox.Elem(func(cur gox.Cursor) error {
		el := comp.Main()
		if el == nil {
			return nil
		}
		p := &modPrinter{mod: mod, cur: cur}
		cur = gox.NewCursor(cur.Context(), p)
		return el(cur)
	}))
}

func (m *modPrinter) printHead(mod gox.Modify, job *gox.JobHeadOpen) error {
	job.Attrs.AddMod(mod)
	return m.cur.Send(job)
}
