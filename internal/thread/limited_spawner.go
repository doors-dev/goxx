package thread

import (
	"context"
	"sync"

	"github.com/gammazero/deque"
)

func newLimitedSpawner(ctx context.Context, limit int) (spawner, context.Context) {
	if limit <= 0 {
		panic("Limit can't be less then 1")
	}
	ctx, cancel := context.WithCancel(ctx)
	return &limitedSpawner{
		hot:  make(chan int, limit),
		pool: make([]chan task, 0, limit),
		cold: new(deque.Deque[task]),
		baseSpawner: baseSpawner{
			ctx:    ctx,
			cancel: cancel,
		},
	}, ctx
}

type limitedSpawner struct {
	mu   sync.Mutex
	hot  chan int
	cold *deque.Deque[task]
	pool []chan task
	baseSpawner
}

func (s *limitedSpawner) sync(t task) {
	if s.ctx.Err() != nil {
		t.done()
		return
	}
	defer t.done()
	err := t.execute()
	s.report(err)
}

func (s *limitedSpawner) spawn(t task) {
	if s.ctx.Err() != nil {
		t.done()
		return
	}
	ok := s.submitHot(t)
	if ok {
		return
	}
	s.submitCold(t)
}

func (s *limitedSpawner) submitCold(t task) {
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		t.done()
		return
	}
	ok := s.submitHot(t)
	if ok {
		s.mu.Unlock()
		return
	}
	count := len(s.pool)
	if count < cap(s.pool) {
		ch := make(chan task)
		s.pool = append(s.pool, ch)
		s.mu.Unlock()
		go s.worker(count, ch)
		select {
		case ch <- t:
		case <-s.ctx.Done():
			t.done()
		}
		return
	}
	s.cold.PushBack(t)
	s.mu.Unlock()
}

func (s *limitedSpawner) submitHot(t task) bool {
	num, ok := s.getHotNum()
	if !ok {
		return false
	}
	select {
	case s.pool[num] <- t:
	case <-s.ctx.Done():
		t.done()
	}
	return true
}

func (s *limitedSpawner) getHotNum() (int, bool) {
	select {
	case num := <-s.hot:
		return num, true
	default:
		return 0, false
	}
}

func (s *limitedSpawner) worker(index int, ch chan task) {
	var err error
	t, ok := s.recv(ch)
	if !ok {
		return
	}
	for {
		err = t.execute()
		s.report(err)
		t.done()
		if err != nil {
			break
		}
		s.mu.Lock()
		if s.ctx.Err() != nil {
			s.mu.Unlock()
			break
		}
		if s.cold.Len() != 0 {
			t = s.cold.PopFront()
			s.mu.Unlock()
			continue
		}
		s.mu.Unlock()
		s.hot <- index
		t, ok = s.recv(ch)
		if !ok {
			break
		}
	}
	s.mu.Lock()
	var queue *deque.Deque[task]
	if s.cold != nil {
		queue = s.cold
		s.cold = nil
	}
	s.mu.Unlock()
	if queue == nil {
		return
	}
	for t := range queue.Iter() {
		t.done()
	}
}

func (s *limitedSpawner) recv(ch chan task) (task, bool) {
	select {
	case <-s.ctx.Done():
		return task{}, false
	case t := <-ch:
		if s.ctx.Err() != nil {
			t.done()
			return task{}, false
		}
		return t, true
	}
}
