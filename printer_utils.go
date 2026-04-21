package goxx

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"github.com/gammazero/deque"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// WriteErr wraps an error returned by the final io.Writer.
//
// It is returned after rendering has succeeded and NewPrinter is draining its
// buffered output to the writer passed to NewPrinter.
type WriteErr struct {
	// Err is the underlying writer error.
	Err error
}

// Error returns the underlying writer error message.
func (we WriteErr) Error() string {
	return we.Err.Error()
}

// Unwrap returns the underlying writer error.
func (we WriteErr) Unwrap() error {
	return we.Err
}

// WriterError returns the underlying writer error from err.
//
// It reports false when err is not, and does not wrap, a WriteErr.
func WriterError(err error) (error, bool) {
	var writeErr WriteErr
	if errors.As(err, &writeErr) {
		return writeErr.Err, true
	}
	return nil, false
}

func newBufferTree(root *deque.Deque[any]) bufferTree {
	return (bufferTree)([]*deque.Deque[any]{root})
}

type bufferTree []*deque.Deque[any]

func (b *bufferTree) Close() error {
main:
	if len(*b) == 0 {
		return nil
	}
	queue := (*b)[len(*b)-1]
	for item := range queue.IterPopFront() {
		switch item := item.(type) {
		case *bytes.Buffer:
			putBuffer(item)
		case *deque.Deque[any]:
			*b = append(*b, item)
			goto main
		default:
			panic("Unexpected item type")
		}
	}
	(*b)[len(*b)-1] = nil
	*b = (*b)[:len(*b)-1]
	if len(*b) != 0 {
		goto main
	}
	return nil
}

var renderOutputReleased = errors.New("goxx: rendered output already released")

func (b *bufferTree) WriteTo(w io.Writer) (n int64, err error) {
main:
	if len(*b) == 0 {
		return 0, renderOutputReleased
	}
	queue := (*b)[len(*b)-1]
	for item := range queue.IterPopFront() {
		switch item := item.(type) {
		case *bytes.Buffer:
			if err != nil {
				putBuffer(item)
				continue
			}
			written, writeErr := item.WriteTo(w)
			putBuffer(item)
			n += written
			if writeErr != nil {
				err = writeErr
			}
		case *deque.Deque[any]:
			*b = append(*b, item)
			goto main
		default:
			panic("Unexpected item type")
		}
	}
	(*b)[len(*b)-1] = nil
	*b = (*b)[:len(*b)-1]
	if len(*b) != 0 {
		goto main
	}
	return
}

// WriterToCloser is buffered printer output returned by Render.
//
// WriteTo writes the buffered output once and releases it. After WriteTo or
// Close, later WriteTo calls return an error. Close releases the output without
// writing it; call Close when you decide not to write the rendered output.
// Close is safe to call more than once.
type WriterToCloser interface {
	io.WriterTo
	io.Closer
}

var _ WriterToCloser = &bufferTree{}
