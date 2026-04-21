package thread

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTimeout = time.Second

func waitForClose(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitForErr(t *testing.T, ch <-chan error, name string) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", name)
		return nil
	}
}

func requireNotClosed(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatalf("%s happened before it should", name)
	default:
	}
}

func requireNotClosedWithin(t *testing.T, ch <-chan struct{}, d time.Duration, name string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatalf("%s happened before it should", name)
	case <-time.After(d):
	}
}

func requireIgnoredTask(t *testing.T, ran *atomic.Bool, message string) {
	t.Helper()
	time.Sleep(25 * time.Millisecond)
	if ran.Load() {
		t.Fatal(message)
	}
}

func newTestTask(spawner spawner, f func(Thread) error) (task, <-chan struct{}) {
	host := newHost()
	th := &thread{
		spawner: spawner,
		host:    host,
	}
	th.counter.Add(1)
	return task{thread: th, fun: f}, host.ch()
}

func TestSubmitRejectsNegativeLimit(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Submit did not panic for a negative limit")
		}
	}()

	_ = Root(context.Background(), -1, func(context.Context, Thread) error {
		return nil
	})
}

func TestSubmitRunsRootAndCancelsContext(t *testing.T) {
	var ran atomic.Bool
	var childCtx context.Context

	err := Root(context.Background(), 1, func(ctx context.Context, _ Thread) error {
		childCtx = ctx
		ran.Store(true)
		return nil
	})
	if err != nil {
		t.Fatalf("Submit() error = %v, want nil", err)
	}
	if !ran.Load() {
		t.Fatal("root did not run")
	}
	waitForClose(t, childCtx.Done(), "derived context cancellation")
}

func TestRootErrorIsReturned(t *testing.T) {
	wantErr := errors.New("root failure")

	err := Root(context.Background(), 1, func(context.Context, Thread) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Submit() error = %v, want %v", err, wantErr)
	}
}

func TestRootPanicIsRecoveredAndReturnedAsError(t *testing.T) {
	err := Root(context.Background(), 1, func(context.Context, Thread) error {
		panic("root boom")
	})
	if err == nil {
		t.Fatal("Submit() error = nil, want panic error")
	}
	if !strings.Contains(err.Error(), "Render panic: root boom") {
		t.Fatalf("Submit() error = %q, want recovered panic", err.Error())
	}
}

func TestSubmitReturnsContextCanceledWhenParentAlreadyCanceled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	rootRan := errors.New("root ran after parent cancellation")

	done := make(chan error, 1)
	go func() {
		done <- Root(parent, 1, func(context.Context, Thread) error {
			return rootRan
		})
	}()

	err := waitForErr(t, done, "Submit after parent cancellation")
	if errors.Is(err, rootRan) {
		t.Fatal("root ran after parent cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit() error = %v, want %v", err, context.Canceled)
	}
}

func TestUnlimitedSubmitReturnsContextCanceledWhenParentAlreadyCanceled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	rootRan := errors.New("root ran after parent cancellation")

	done := make(chan error, 1)
	go func() {
		done <- Root(parent, 0, func(context.Context, Thread) error {
			return rootRan
		})
	}()

	err := waitForErr(t, done, "unlimited Submit after parent cancellation")
	if errors.Is(err, rootRan) {
		t.Fatal("root ran after parent cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit() error = %v, want %v", err, context.Canceled)
	}
}

func TestInternalLimitedSpawnerRejectsNonPositiveLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		t.Run("limit "+strconv.Itoa(limit), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("newLimitedSpawner did not panic for limit %d", limit)
				}
			}()

			_, _ = newLimitedSpawner(context.Background(), limit)
		})
	}
}

func TestLimitedSpawnerIgnoresSpawnAfterShutdown(t *testing.T) {
	sp, _ := newLimitedSpawner(context.Background(), 1)
	sp.shutdown()

	var ran atomic.Bool
	tk, done := newTestTask(sp, func(Thread) error {
		ran.Store(true)
		return nil
	})
	sp.spawn(tk)

	waitForClose(t, done, "task release after limited spawner shutdown")
	if ran.Load() {
		t.Fatal("task ran after limited spawner shutdown")
	}
}

func TestLimitedSubmitColdAfterShutdownReleasesTask(t *testing.T) {
	sp, _ := newLimitedSpawner(context.Background(), 1)
	s := sp.(*limitedSpawner)
	s.shutdown()

	var ran atomic.Bool
	tk, done := newTestTask(sp, func(Thread) error {
		ran.Store(true)
		return nil
	})
	s.submitCold(tk)

	waitForClose(t, done, "cold task release after limited spawner shutdown")
	if ran.Load() {
		t.Fatal("cold task ran after limited spawner shutdown")
	}
}

func TestLimitedSubmitHotAfterShutdownReleasesTask(t *testing.T) {
	sp, _ := newLimitedSpawner(context.Background(), 1)
	s := sp.(*limitedSpawner)
	s.pool = append(s.pool, make(chan task))
	s.hot <- 0
	s.shutdown()

	var ran atomic.Bool
	tk, done := newTestTask(sp, func(Thread) error {
		ran.Store(true)
		return nil
	})
	if !s.submitHot(tk) {
		t.Fatal("submitHot() = false, want true")
	}

	waitForClose(t, done, "hot task release after limited spawner shutdown")
	if ran.Load() {
		t.Fatal("hot task ran after limited spawner shutdown")
	}
}

func TestUnlimitedSpawnerExecuteAfterShutdownReleasesTask(t *testing.T) {
	sp, _ := newUnlimitedSpawner(context.Background())
	s := sp.(*unlimitedSpawner)
	s.shutdown()

	var ran atomic.Bool
	tk, done := newTestTask(sp, func(Thread) error {
		ran.Store(true)
		return nil
	})
	s.execute(tk)

	waitForClose(t, done, "task release after unlimited spawner shutdown")
	if ran.Load() {
		t.Fatal("task ran after unlimited spawner shutdown")
	}
}

func TestLimitedGoIsNonBlockingWhenCalledFromRunningTaskAtLimit(t *testing.T) {
	outerStarted := make(chan struct{})
	releaseOuter := make(chan struct{})
	nestedGoReturned := make(chan struct{})
	nestedRan := make(chan struct{})

	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() {
			close(releaseOuter)
		})
	})

	err := Root(context.Background(), 1, func(_ context.Context, th Thread) error {
		th.Go(func(th Thread) error {
			close(outerStarted)
			th.Go(func(Thread) error {
				close(nestedRan)
				return nil
			})
			close(nestedGoReturned)
			<-releaseOuter
			return nil
		})
		waitForClose(t, outerStarted, "outer task start")
		waitForClose(t, nestedGoReturned, "nested Go return")
		requireNotClosedWithin(t, nestedRan, 25*time.Millisecond, "nested task start while limit is full")
		releaseOnce.Do(func() {
			close(releaseOuter)
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Submit() error = %v, want nil", err)
	}
	waitForClose(t, nestedRan, "nested task run")
}

func TestLimitedQueueDrainsAfterCapacityOpens(t *testing.T) {
	release := make(chan struct{})
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	thirdRan := make(chan struct{})
	fourthRan := make(chan struct{})

	err := Root(context.Background(), 2, func(_ context.Context, th Thread) error {
		th.Go(func(Thread) error {
			close(firstStarted)
			<-release
			return nil
		})
		th.Go(func(Thread) error {
			close(secondStarted)
			<-release
			return nil
		})

		waitForClose(t, firstStarted, "first task start")
		waitForClose(t, secondStarted, "second task start")

		th.Go(func(Thread) error {
			close(thirdRan)
			return nil
		})
		th.Go(func(Thread) error {
			close(fourthRan)
			return nil
		})

		requireNotClosed(t, thirdRan, "third task start while limit is full")
		requireNotClosed(t, fourthRan, "fourth task start while limit is full")
		requireNotClosedWithin(t, thirdRan, 25*time.Millisecond, "third task start while limit is full")
		requireNotClosed(t, fourthRan, "fourth task start while limit is full")

		close(release)
		return nil
	})
	if err != nil {
		t.Fatalf("Submit() error = %v, want nil", err)
	}
	waitForClose(t, thirdRan, "third queued task run")
	waitForClose(t, fourthRan, "fourth queued task run")
}

func TestUnlimitedSpawnerRunsSpawnedTasks(t *testing.T) {
	firstRan := make(chan struct{})
	secondRan := make(chan struct{})

	err := Root(context.Background(), 0, func(_ context.Context, th Thread) error {
		th.Go(func(Thread) error {
			close(firstRan)
			return nil
		})
		th.Go(func(Thread) error {
			close(secondRan)
			return nil
		})
		return nil
	})
	if err != nil {
		t.Fatalf("Submit() error = %v, want nil", err)
	}
	waitForClose(t, firstRan, "first unlimited task run")
	waitForClose(t, secondRan, "second unlimited task run")
}

func TestUnlimitedTaskErrorCancelsContextAndIsReturned(t *testing.T) {
	wantErr := errors.New("unlimited failure")
	var childCtx context.Context

	err := Root(context.Background(), 0, func(ctx context.Context, th Thread) error {
		childCtx = ctx
		th.Go(func(Thread) error {
			return wantErr
		})
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Submit() error = %v, want %v", err, wantErr)
	}
	waitForClose(t, childCtx.Done(), "derived context cancellation")
}

func TestFirstErrorCancelsContextSkipsQueuedWorkAndIsReturned(t *testing.T) {
	wantErr := errors.New("first failure")
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	queuedRan := make(chan struct{})
	var childCtx context.Context

	err := Root(context.Background(), 1, func(ctx context.Context, th Thread) error {
		childCtx = ctx
		th.Go(func(Thread) error {
			close(firstStarted)
			<-releaseFirst
			return wantErr
		})
		waitForClose(t, firstStarted, "first task start")

		th.Go(func(Thread) error {
			close(queuedRan)
			return nil
		})

		close(releaseFirst)
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Submit() error = %v, want %v", err, wantErr)
	}
	waitForClose(t, childCtx.Done(), "derived context cancellation")
	requireNotClosedWithin(t, queuedRan, 25*time.Millisecond, "queued task run after error")
}

func TestQueuedTaskErrorIsReturned(t *testing.T) {
	wantErr := errors.New("queued failure")
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	queuedStarted := make(chan struct{})
	var childCtx context.Context

	err := Root(context.Background(), 1, func(ctx context.Context, th Thread) error {
		childCtx = ctx
		th.Go(func(Thread) error {
			close(firstStarted)
			<-releaseFirst
			return nil
		})
		waitForClose(t, firstStarted, "first task start")

		th.Go(func(Thread) error {
			close(queuedStarted)
			return wantErr
		})
		requireNotClosed(t, queuedStarted, "queued task start while limit is full")

		close(releaseFirst)
		return nil
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Submit() error = %v, want %v", err, wantErr)
	}
	waitForClose(t, queuedStarted, "queued task start")
	waitForClose(t, childCtx.Done(), "derived context cancellation")
}

func TestFirstReportedErrorWins(t *testing.T) {
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	releaseSecond := make(chan struct{})
	var childCtx context.Context

	err := Root(context.Background(), 2, func(ctx context.Context, th Thread) error {
		childCtx = ctx
		th.Go(func(Thread) error {
			close(firstStarted)
			<-releaseFirst
			return firstErr
		})
		th.Go(func(Thread) error {
			close(secondStarted)
			<-releaseSecond
			return secondErr
		})

		waitForClose(t, firstStarted, "first task start")
		waitForClose(t, secondStarted, "second task start")
		close(releaseFirst)
		waitForClose(t, childCtx.Done(), "group cancellation after first error")
		close(releaseSecond)
		return nil
	})
	if !errors.Is(err, firstErr) {
		t.Fatalf("Submit() error = %v, want %v", err, firstErr)
	}
}

func TestPanicIsRecoveredAndReturnedAsError(t *testing.T) {
	var childCtx context.Context

	err := Root(context.Background(), 1, func(ctx context.Context, th Thread) error {
		childCtx = ctx
		th.Go(func(Thread) error {
			panic("boom")
		})
		return nil
	})
	if err == nil {
		t.Fatal("Submit() error = nil, want panic error")
	}
	if !strings.Contains(err.Error(), "Render panic: boom") {
		t.Fatalf("Submit() error = %q, want recovered panic", err.Error())
	}
	waitForClose(t, childCtx.Done(), "derived context cancellation")
}

func TestGoAfterParentCancellationIsIgnoredAndReturned(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	var ran atomic.Bool

	err := Root(parent, 1, func(_ context.Context, th Thread) error {
		cancel()
		th.Go(func(Thread) error {
			ran.Store(true)
			return nil
		})
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit() error = %v, want %v", err, context.Canceled)
	}
	requireIgnoredTask(t, &ran, "task ran after parent cancellation")
}

func TestParentCancellationSkipsQueuedWorkAndIsReturned(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	queuedRan := make(chan struct{})

	err := Root(parent, 1, func(_ context.Context, th Thread) error {
		th.Go(func(Thread) error {
			close(firstStarted)
			<-releaseFirst
			return nil
		})
		waitForClose(t, firstStarted, "first task start")

		th.Go(func(Thread) error {
			close(queuedRan)
			return nil
		})

		cancel()
		close(releaseFirst)
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Submit() error = %v, want %v", err, context.Canceled)
	}
	requireNotClosedWithin(t, queuedRan, 25*time.Millisecond, "queued task run after parent cancellation")
}

func TestGoAfterCallbackReturnsPanics(t *testing.T) {
	var saved Thread

	err := Root(context.Background(), 1, func(_ context.Context, th Thread) error {
		saved = th
		return nil
	})
	if err != nil {
		t.Fatalf("Submit() error = %v, want nil", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Go after callback returned did not panic")
		}
		err, ok := r.(scheduleError)
		if !ok {
			t.Fatalf("Go after callback returned panic = %T, want scheduleError", r)
		}
		if got, want := err.Error(), "thread is used after consumption"; got != want {
			t.Fatalf("scheduleError.Error() = %q, want %q", got, want)
		}
	}()
	saved.Go(func(Thread) error {
		return nil
	})
}
