package goxx_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
)

const testTimeout = time.Second

type renderResult struct {
	html string
	err  error
}

func printElem(w io.Writer, el gox.Elem, opts ...goxx.Option) error {
	return el.Print(context.Background(), goxx.NewPrinter(w, opts...))
}

func renderString(el gox.Elem, opts ...goxx.Option) (string, error) {
	var b strings.Builder
	err := printElem(&b, el, opts...)
	return b.String(), err
}

type workerVariant struct {
	name string
	opts []goxx.Option
}

func forEachWorkerVariant(t *testing.T, fn func(t *testing.T, variant workerVariant)) {
	t.Helper()
	variants := []workerVariant{
		{name: "default"},
		{name: "unlimited", opts: []goxx.Option{goxx.OptionWorkers(0)}},
		{name: "one_worker", opts: []goxx.Option{goxx.OptionWorkers(1)}},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			fn(t, variant)
		})
	}
}

func withOptions(variant workerVariant, opts ...goxx.Option) []goxx.Option {
	merged := make([]goxx.Option, 0, len(variant.opts)+len(opts))
	merged = append(merged, variant.opts...)
	merged = append(merged, opts...)
	return merged
}

func waitForClose(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func waitForRender(t *testing.T, ch <-chan renderResult) renderResult {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for render")
		return renderResult{}
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

func TestNewPrinterRejectsNegativeWorkerCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewPrinter did not panic for a negative worker count")
		}
	}()

	_ = goxx.NewPrinter(io.Discard, goxx.OptionWorkers(-1))
}

func TestNewPrinterDelegatesRootNonComponentJob(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		var b strings.Builder
		printer := goxx.NewPrinter(&b, variant.opts...)

		err := printer.Send(gox.NewJobText(context.Background(), "<escaped>"))
		if err != nil {
			t.Fatalf("Send() error = %v, want nil", err)
		}
		const want = "&lt;escaped&gt;"
		if got := b.String(); got != want {
			t.Fatalf("Send() output = %q, want %q", got, want)
		}
	})
}

func TestParallelProxyIgnoresNilElem(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("before|"); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, nil); err != nil {
				return err
			}
			return cur.Text("after|")
		})

		got, err := renderString(root, variant.opts...)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}
		const want = "before|after|"
		if got != want {
			t.Fatalf("Print() html = %q, want %q", got, want)
		}
	})
}

func TestParallelProxyOutsideNewPrinterRendersSequentially(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		if err := cur.Text("before|"); err != nil {
			return err
		}
		if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			return cur.Text("branch|")
		})); err != nil {
			return err
		}
		return cur.Text("after|")
	})

	var b strings.Builder
	err := root.Print(context.Background(), gox.NewPrinter(&b))
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	const want = "before|branch|after|"
	if got := b.String(); got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func TestParallelPrinterNilRootComponentRendersNothing(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		var b strings.Builder
		printer := goxx.NewPrinter(&b, variant.opts...)

		err := printer.Send(gox.NewJobComp(context.Background(), nilComp{}))
		if err != nil {
			t.Fatalf("Send() error = %v, want nil", err)
		}
		if got := b.String(); got != "" {
			t.Fatalf("Send() output = %q, want empty", got)
		}
	})
}

func TestParallelPrinterCanceledContextWritesNothing(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var b strings.Builder
		root := gox.Elem(func(cur gox.Cursor) error {
			return cur.Text("should not render")
		})

		err := root.Print(ctx, goxx.NewPrinter(&b, variant.opts...))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Print() error = %v, want %v", err, context.Canceled)
		}
		if got := b.String(); got != "" {
			t.Fatalf("Print() output = %q, want empty", got)
		}
	})
}

func TestParallelPrinterRendersTextBeforeFirstParallelBranch(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("prefix|"); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				return cur.Text("branch|")
			})); err != nil {
				return err
			}
			return cur.Text("suffix|")
		})

		got, err := renderString(root, variant.opts...)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}
		const want = "prefix|branch|suffix|"
		if got != want {
			t.Fatalf("Print() html = %q, want %q", got, want)
		}
	})
}

func TestParallelPrinterPreservesBranchOrder(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		slowStarted := make(chan struct{})
		releaseSlow := make(chan struct{})
		fastDone := make(chan struct{})

		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("before|"); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				close(slowStarted)
				<-releaseSlow
				return cur.Text("slow|")
			})); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				if err := cur.Text("fast|"); err != nil {
					return err
				}
				close(fastDone)
				return nil
			})); err != nil {
				return err
			}
			return cur.Text("after|")
		})

		done := make(chan renderResult, 1)
		go func() {
			html, err := renderString(root, variant.opts...)
			done <- renderResult{html: html, err: err}
		}()

		waitForClose(t, slowStarted, "slow branch start")
		if variant.name == "one_worker" {
			requireNotClosedWithin(t, fastDone, 25*time.Millisecond, "fast branch finish while one worker is busy")
		} else {
			waitForClose(t, fastDone, "fast branch finish")
		}
		close(releaseSlow)

		result := waitForRender(t, done)
		if result.err != nil {
			t.Fatalf("Print() error = %v, want nil", result.err)
		}
		const want = "before|slow|fast|after|"
		if result.html != want {
			t.Fatalf("Print() html = %q, want %q", result.html, want)
		}
	})
}

func TestParallelPrinterHonorsWorkerLimit(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})

	root := gox.Elem(func(cur gox.Cursor) error {
		if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			close(firstStarted)
			<-releaseFirst
			return cur.Text("first|")
		})); err != nil {
			return err
		}
		if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			close(secondStarted)
			return cur.Text("second|")
		})); err != nil {
			return err
		}
		return nil
	})

	done := make(chan renderResult, 1)
	go func() {
		html, err := renderString(root, goxx.OptionWorkers(1))
		done <- renderResult{html: html, err: err}
	}()

	waitForClose(t, firstStarted, "first branch start")
	requireNotClosedWithin(t, secondStarted, 25*time.Millisecond, "second branch start while worker is busy")
	close(releaseFirst)

	result := waitForRender(t, done)
	if result.err != nil {
		t.Fatalf("Print() error = %v, want nil", result.err)
	}
	waitForClose(t, secondStarted, "second branch start")
	const want = "first|second|"
	if result.html != want {
		t.Fatalf("Print() html = %q, want %q", result.html, want)
	}
}

func TestParallelPrinterZeroWorkersRunsBranchesWithoutPoolLimit(t *testing.T) {
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondStarted := make(chan struct{})

	root := gox.Elem(func(cur gox.Cursor) error {
		if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			close(firstStarted)
			<-releaseFirst
			return cur.Text("first|")
		})); err != nil {
			return err
		}
		if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			close(secondStarted)
			return cur.Text("second|")
		})); err != nil {
			return err
		}
		return nil
	})

	done := make(chan renderResult, 1)
	go func() {
		html, err := renderString(root, goxx.OptionWorkers(0))
		done <- renderResult{html: html, err: err}
	}()

	waitForClose(t, firstStarted, "first branch start")
	waitForClose(t, secondStarted, "second branch start")
	close(releaseFirst)

	result := waitForRender(t, done)
	if result.err != nil {
		t.Fatalf("Print() error = %v, want nil", result.err)
	}
	const want = "first|second|"
	if result.html != want {
		t.Fatalf("Print() html = %q, want %q", result.html, want)
	}
}

func TestParallelPrinterRenderTimeReflectsWorkerSaturation(t *testing.T) {
	const branches = 10
	const delay = 30 * time.Millisecond

	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		root := gox.Elem(func(cur gox.Cursor) error {
			for i := range branches {
				i := i
				if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
					<-time.After(delay)
					return cur.Text(fmt.Sprintf("%02d|", i))
				})); err != nil {
					return err
				}
			}
			return nil
		})

		start := time.Now()
		got, err := renderString(root, variant.opts...)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}

		want := strings.Builder{}
		for i := range branches {
			fmt.Fprintf(&want, "%02d|", i)
		}
		if got != want.String() {
			t.Fatalf("Print() html = %q, want %q", got, want.String())
		}

		workers := 7
		switch variant.name {
		case "unlimited":
			workers = branches
		case "one_worker":
			workers = 1
		}
		batches := (branches + workers - 1) / workers
		minElapsed := time.Duration(batches) * delay
		if elapsed < minElapsed-10*time.Millisecond {
			t.Fatalf("Print() elapsed = %v, want at least about %v for %d branches with %d workers", elapsed, minElapsed, branches, workers)
		}
	})
}

func TestOptionPrinterReceivesComponentJobsUnlessFlat(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		child := gox.Elem(func(cur gox.Cursor) error {
			return cur.Text("child")
		})
		root := gox.Elem(func(cur gox.Cursor) error {
			return cur.Comp(child)
		})

		var normalCompJobs atomic.Int32
		got, err := renderString(root, withOptions(variant, goxx.OptionPrinter(countCompPrinter(&normalCompJobs)))...)
		if err != nil {
			t.Fatalf("Print() with custom printer error = %v, want nil", err)
		}
		if got != "child" {
			t.Fatalf("Print() with custom printer html = %q, want child", got)
		}
		if normalCompJobs.Load() != 1 {
			t.Fatalf("custom printer saw %d component jobs, want 1", normalCompJobs.Load())
		}

		var flatCompJobs atomic.Int32
		got, err = renderString(root, withOptions(variant, goxx.OptionFlat(), goxx.OptionPrinter(countCompPrinter(&flatCompJobs)))...)
		if err != nil {
			t.Fatalf("Print() with flat custom printer error = %v, want nil", err)
		}
		if got != "child" {
			t.Fatalf("Print() with flat custom printer html = %q, want child", got)
		}
		if flatCompJobs.Load() != 0 {
			t.Fatalf("flat custom printer saw %d component jobs, want 0", flatCompJobs.Load())
		}
	})
}

func TestParallelPrinterRendersHTMLAroundParallelBranch(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Init("main"); err != nil {
				return err
			}
			if err := cur.AttrSet("id", "root"); err != nil {
				return err
			}
			if err := cur.Submit(); err != nil {
				return err
			}
			if err := cur.Text("a"); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				if err := cur.Init("span"); err != nil {
					return err
				}
				if err := cur.Submit(); err != nil {
					return err
				}
				if err := cur.Text("b"); err != nil {
					return err
				}
				return cur.Close()
			})); err != nil {
				return err
			}
			if err := cur.Text("c"); err != nil {
				return err
			}
			return cur.Close()
		})

		got, err := renderString(root, variant.opts...)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}
		const want = `<main id="root">a<span>b</span>c</main>`
		if got != want {
			t.Fatalf("Print() html = %q, want %q", got, want)
		}
	})
}

func TestParallelPrinterPreservesNestedParallelBranches(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		nestedBranch := func(prefix string) gox.Elem {
			return gox.Elem(func(cur gox.Cursor) error {
				if err := cur.Text(prefix + "0|"); err != nil {
					return err
				}
				if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
					<-time.After(5 * time.Millisecond)
					return cur.Text(prefix + "1|")
				})); err != nil {
					return err
				}
				if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
					return cur.Text(prefix + "2|")
				})); err != nil {
					return err
				}
				return cur.Text(prefix + "3|")
			})
		}

		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("root|"); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, nestedBranch("a")); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, nestedBranch("b")); err != nil {
				return err
			}
			return cur.Text("tail|")
		})

		got, err := renderString(root, variant.opts...)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}
		const want = "root|a0|a1|a2|a3|b0|b1|b2|b3|tail|"
		if got != want {
			t.Fatalf("Print() html = %q, want %q", got, want)
		}
	})
}

func TestParallelPrinterRendersRegularComponent(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		child := gox.Elem(func(cur gox.Cursor) error {
			return cur.Text("component|")
		})
		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("before|"); err != nil {
				return err
			}
			if err := cur.Comp(child); err != nil {
				return err
			}
			return cur.Text("after|")
		})

		got, err := renderString(root, variant.opts...)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}
		const want = "before|component|after|"
		if got != want {
			t.Fatalf("Print() html = %q, want %q", got, want)
		}
	})
}

func TestParallelPrinterFlatOptionAppliesInsideParallelBranch(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		slowStarted := make(chan struct{})
		releaseSlow := make(chan struct{})
		fastDone := make(chan struct{})

		child := gox.Elem(func(cur gox.Cursor) error {
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				close(slowStarted)
				<-releaseSlow
				return cur.Text("slow|")
			})); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				if err := cur.Text("fast|"); err != nil {
					return err
				}
				close(fastDone)
				return nil
			})); err != nil {
				return err
			}
			return cur.Text("tail|")
		})
		root := gox.Elem(func(cur gox.Cursor) error {
			return goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				return cur.Comp(child)
			}))
		})

		done := make(chan renderResult, 1)
		go func() {
			html, err := renderString(root, withOptions(variant, goxx.OptionFlat())...)
			done <- renderResult{html: html, err: err}
		}()

		waitForClose(t, slowStarted, "slow nested branch start")
		if variant.name == "one_worker" {
			requireNotClosedWithin(t, fastDone, 25*time.Millisecond, "fast nested branch finish while one worker is busy")
		} else {
			waitForClose(t, fastDone, "fast nested branch finish")
		}
		close(releaseSlow)

		result := waitForRender(t, done)
		if result.err != nil {
			t.Fatalf("Print() error = %v, want nil", result.err)
		}
		const want = "slow|fast|tail|"
		if result.html != want {
			t.Fatalf("Print() html = %q, want %q", result.html, want)
		}
	})
}

func TestParallelPrinterReturnsParallelBranchError(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		wantErr := errors.New("parallel failed")
		root := gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("before|"); err != nil {
				return err
			}
			if err := goxx.Parallel().Proxy(cur, gox.Elem(func(gox.Cursor) error {
				return wantErr
			})); err != nil {
				return err
			}
			return cur.Text("after|")
		})

		got, err := renderString(root, variant.opts...)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Print() error = %v, want %v", err, wantErr)
		}
		if got != "" {
			t.Fatalf("Print() html = %q, want no partial output after error", got)
		}
	})
}

func TestParallelPrinterReturnsWriterError(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		wantErr := errors.New("write failed")
		root := gox.Elem(func(cur gox.Cursor) error {
			return cur.Text("hello")
		})

		err := printElem(errorWriter{err: wantErr}, root, variant.opts...)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Print() error = %v, want %v", err, wantErr)
		}
		if got, ok := goxx.WriterError(err); !ok || !errors.Is(got, wantErr) {
			t.Fatalf("WriterError() = %v, %v, want %v, true", got, ok, wantErr)
		}
		if got := err.Error(); got != wantErr.Error() {
			t.Fatalf("Print() error string = %q, want %q", got, wantErr.Error())
		}
	})
}

func TestWriterErrorRecognizesWrappedWriteErr(t *testing.T) {
	wantErr := errors.New("write failed")
	err := errors.Join(goxx.WriteErr{Err: wantErr})

	got, ok := goxx.WriterError(err)
	if !ok {
		t.Fatal("WriterError() ok = false, want true")
	}
	if !errors.Is(got, wantErr) {
		t.Fatalf("WriterError() error = %v, want %v", got, wantErr)
	}
}

func TestWriterErrorReturnsFalseForRenderError(t *testing.T) {
	got, ok := goxx.WriterError(errors.New("render failed"))
	if ok {
		t.Fatalf("WriterError() = %v, true, want nil, false", got)
	}
	if got != nil {
		t.Fatalf("WriterError() error = %v, want nil", got)
	}
}

type nilComp struct{}

func (nilComp) Main() gox.Elem {
	return nil
}

type countingPrinter struct {
	next     gox.Printer
	compJobs *atomic.Int32
}

func countCompPrinter(compJobs *atomic.Int32) func(io.Writer) gox.Printer {
	return func(w io.Writer) gox.Printer {
		return countingPrinter{
			next:     gox.NewPrinter(w),
			compJobs: compJobs,
		}
	}
}

func (p countingPrinter) Send(j gox.Job) error {
	if _, ok := j.(*gox.JobComp); ok {
		p.compJobs.Add(1)
	}
	return p.next.Send(j)
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
