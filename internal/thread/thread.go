package thread

import (
	"sync/atomic"
)

type scheduleError struct{}

func (scheduleError) Error() string {
	return "thread is used after consumption"
}

var shedulePanic = scheduleError{}

type Thread = *thread

type thread struct {
	spawner spawner
	host    host
	counter atomic.Int32
}

func (t Thread) Go(f func(t Thread) error) {
	count := t.counter.Add(1)
	if count == 1 {
		panic(shedulePanic)
	}
	n := &thread{
		host:    t,
		spawner: t.spawner,
	}
	n.counter.Add(1)
	t.spawner.spawn(task{n, f})
}

func (t *thread) root(f func(t *thread) error) {
	t.counter.Add(1)
	t.spawner.sync(task{t, f})
}

func (t *thread) done() {
	count := t.counter.Add(-1)
	if count != 0 {
		return
	}
	t.host.done()
}
