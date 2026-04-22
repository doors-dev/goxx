package hx

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/doors-dev/goxx"
)

var prefix = "/~/"

// SetPrefix changes the URL prefix used by generated hx-* attributes and
// Handler.
//
// The default prefix is "/~/". SetPrefix trims leading and trailing slashes and
// stores the prefix with surrounding slashes, so SetPrefix("/hx") makes Prefix
// return "/hx/". Call SetPrefix before rendering pages or serving fragments.
func SetPrefix(p string) {
	p = strings.Trim(p, "/")
	if p == "" {
		panic("hx: prefix must not be empty")
	}
	if p != url.PathEscape(p) {
		panic(fmt.Sprintf("hx: prefix %q is not URL path safe", p))
	}
	prefix = "/" + p + "/"
}

// Prefix returns the URL prefix used for generated hx-* attributes and Handler.
func Prefix() string {
	return prefix
}

var options = []goxx.Option{}

// SetOptions sets the goxx.Render options used when Handler renders fragments.
func SetOptions(opts ...goxx.Option) {
	options = opts
}
