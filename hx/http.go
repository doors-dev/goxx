package hx

import "net/http"

// Responser exposes HTTP response controls to fragment handlers.
//
// Fragment handlers return markup instead of writing the body directly, but
// they can still set headers, cookies, and the success status before the
// rendered fragment is written.
type Responser interface {
	// Header returns the response headers that will be sent with the fragment.
	Header() http.Header

	// SetStatus sets the HTTP status code used after rendering succeeds.
	SetStatus(statusCode int)

	// SetCookie adds cookie to the response.
	SetCookie(cookie *http.Cookie)
}

type responser struct {
	w      http.ResponseWriter
	status int
}

func (h *responser) SetCookie(cookie *http.Cookie) {
	http.SetCookie(h.w, cookie)
}

func (h *responser) SetStatus(statusCode int) {
	h.status = statusCode
}

func (h *responser) Header() http.Header {
	return h.w.Header()
}

var _ Responser = (*responser)(nil)
