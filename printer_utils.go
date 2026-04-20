package goxx

import (
	"bytes"
	"errors"
	"sync"
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
