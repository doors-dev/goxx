package hx

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/doors-dev/gox"
)

// HandlerFunc renders an HTMX fragment response.
//
// Handler functions must be named top-level package functions so Register can
// derive a stable endpoint ID from the runtime function name.
type HandlerFunc = func(w Responser, r *http.Request) gox.Elem

// Get returns an attribute modifier that adds hx-get for h.
func Get(h func(w Responser, r *http.Request) gox.Elem) gox.Modify {
	return newMod(http.MethodGet, h)
}

// Post returns an attribute modifier that adds hx-post for h.
func Post(h func(w Responser, r *http.Request) gox.Elem) gox.Modify {
	return newMod(http.MethodPost, h)
}

// Put returns an attribute modifier that adds hx-put for h.
func Put(h func(w Responser, r *http.Request) gox.Elem) gox.Modify {
	return newMod(http.MethodPut, h)
}

// Patch returns an attribute modifier that adds hx-patch for h.
func Patch(h func(w Responser, r *http.Request) gox.Elem) gox.Modify {
	return newMod(http.MethodPatch, h)
}

// Delete returns an attribute modifier that adds hx-delete for h.
func Delete(h func(w Responser, r *http.Request) gox.Elem) gox.Modify {
	return newMod(http.MethodDelete, h)
}

func newMod(m string, h HandlerFunc) gox.Modify {
	return gox.ModifyFunc(func(_ context.Context, _ string, attrs gox.Attrs) error {
		name := "hx-" + strings.ToLower(m)
		id, err := Register(h)
		if err != nil {
			return fmt.Errorf("hx: register %s handler: %w", name, err)
		}
		attrs.Get(name).Set(prefix + id)
		return nil
	})
}
