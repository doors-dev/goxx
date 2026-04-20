package thread

import "context"

func newUnlimitedSpawner(ctx context.Context) (spawner, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &unlimitedSpawner{
		baseSpawner: baseSpawner{
			ctx:    ctx,
			cancel: cancel,
		},
	}, ctx
}

type unlimitedSpawner struct {
	baseSpawner
}

func (s *unlimitedSpawner) sync(t task) {
	if s.ctx.Err() != nil {
		t.done()
		return
	}
	defer t.done()
	err := t.execute()
	s.report(err)
}

func (s *unlimitedSpawner) spawn(t task) {
	go s.execute(t)
}

func (s *unlimitedSpawner) execute(t task) {
	defer t.done()
	if s.ctx.Err() != nil {
		return
	}
	err := t.execute()
	s.report(err)
}

var _ spawner = &unlimitedSpawner{}
