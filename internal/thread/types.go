package thread

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type fun = func(t *thread) error

type host interface {
	done()
}

type task struct {
	thread *thread
	fun    fun
}

func (t task) execute() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if r == shedulePanic {
				panic(r)
			}
			err = fmt.Errorf("Render panic: %v\n%s", r, debug.Stack())
		}
	}()
	err = t.fun(t.thread)
	return
}

func (t task) done() {
	t.thread.done()
}

type spawner interface {
	err() error
	sync(task)
	spawn(task)
	shutdown()
}

type baseSpawner struct {
	ctx     context.Context
	cancel  context.CancelFunc
	errGuar sync.Once
	error   error
}

func (g *baseSpawner) shutdown() {
	g.cancel()
}

func (g *baseSpawner) report(err error) {
	if err == nil {
		return
	}
	g.errGuar.Do(func() {
		g.error = err
	})
	g.cancel()
}

func (g *baseSpawner) err() error {
	return g.error
}
