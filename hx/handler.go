package hx

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/doors-dev/goxx"
	"github.com/mr-tron/base58/base58"
	"github.com/zeebo/xxh3"
)

// Handler serves registered HTMX fragment handlers.
//
// Mount it under Prefix. For example, the standard library ServeMux can use
// mux.HandleFunc(hx.Prefix(), hx.Handler), while routers with path variables
// usually mount hx.Prefix()+"{id}".
func Handler(w http.ResponseWriter, r *http.Request) {
	id, ok := strings.CutPrefix(r.URL.Path, prefix)
	if !ok {
		slog.Warn(
			"hx: request path does not match handler prefix",
			"method", r.Method,
			"path", r.URL.Path,
			"prefix", prefix,
		)
		http.NotFound(w, r)
		return
	}
	entry, ok := handlers.Load(id)
	if !ok {
		slog.Warn(
			"hx: no handler registered for request",
			"method", r.Method,
			"path", r.URL.Path,
			"id", id,
		)
		http.NotFound(w, r)
		return
	}
	handler := entry.(HandlerFunc)
	resp := &responser{w: w, status: http.StatusOK}
	elem := handler(resp, r)
	wc, err := goxx.Render(r.Context(), elem, options...)
	if err != nil {
		slog.Error(
			"hx: render handler failed",
			"method", r.Method,
			"path", r.URL.Path,
			"id", id,
			"error", err,
		)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(resp.status)
	if _, err := wc.WriteTo(w); err != nil {
		slog.Debug(
			"hx: response write stopped before completion",
			"method", r.Method,
			"path", r.URL.Path,
			"id", id,
			"error", err,
		)
	}
}

var handlers sync.Map

// MustRegister registers handler and returns its stable endpoint ID.
//
// It panics if handler is nil or is not a named top-level package function.
// MustRegister is useful during package init in load-balanced applications,
// where every instance should know every fragment endpoint before serving
// traffic.
func MustRegister(handler HandlerFunc) string {
	id, err := Register(handler)
	if err != nil {
		panic(err)
	}
	return id
}

// Register registers handler and returns its stable endpoint ID.
//
// The ID is derived from the handler's runtime function name. Register accepts
// only named top-level package functions; closures and method values are
// rejected to keep the registry bounded. Re-registering the same handler reuses
// the existing endpoint.
func Register(handler HandlerFunc) (string, error) {
	if handler == nil {
		return "", errors.New("hx: handler must not be nil")
	}
	name, ok := funName(handler)
	if !ok {
		return "", errors.New("hx: handler must be a named top-level package function; closures and method values are not supported")
	}
	hash := xxh3.New()
	hash.WriteString(name)
	output := hash.Sum128().Bytes()
	id := base58.Encode(output[:])
	handlers.LoadOrStore(id, handler)
	return id, nil
}
