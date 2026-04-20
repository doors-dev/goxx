package goxx_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
)

func ExampleNewPrinter_errorHandling() {
	page := gox.Elem(func(cur gox.Cursor) error {
		return cur.Text("ok")
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := page.Print(r.Context(), goxx.NewPrinter(w))
		if err == nil {
			return
		}

		if err, ok := goxx.WriterError(err); ok {
			slog.Warn("response write failed", "err", err)
			return
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.Debug("render stopped before completion", "err", err)
			return
		}

		http.Error(w, "render failed", http.StatusInternalServerError)
	})

	_ = handler
}
