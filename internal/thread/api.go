package thread

import (
	"context"
)

func Root(ctx context.Context, limit int, root func(ctx context.Context, t Thread) error) error {
	if limit < 0 {
		panic("Limit can't be less than 0")
	}
	var spawner spawner
	if limit == 0 {
		spawner, ctx = newUnlimitedSpawner(ctx)
	} else {
		spawner, ctx = newLimitedSpawner(ctx, limit)
	}
	host := newHost()
	t := &thread{
		spawner: spawner,
		host:    host,
	}
	t.root(func(t *thread) error {
		return root(ctx, t)
	})
	host.wait()
	err := ctx.Err()
	spawner.shutdown()
	if err := spawner.err(); err != nil {
		return err
	}
	return err
}

func newHost() rootHost {
	return make(chan struct{})
}

type rootHost chan struct{}

func (r rootHost) wait() {
	<-r.ch()
}

func (r rootHost) ch() chan struct{} {
	return (chan struct{})(r)
}

func (r rootHost) done() {
	close(r.ch())
}
