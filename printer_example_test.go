package goxx_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
)

func ExampleRender_httpHandler() {
	page := gox.Elem(func(cur gox.Cursor) error {
		return cur.Text("ok")
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out, err := goxx.Render(r.Context(), page)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.Debug("render stopped before completion", "err", err)
			return
		}
		if err != nil {
			http.Error(w, "render failed", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

		if _, err := out.WriteTo(w); err != nil {
			slog.Warn("response write failed", "err", err)
		}
	})

	_ = handler
}
