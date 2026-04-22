// Package hx provides HTMX helpers for GoX templates.
//
// The verb helpers, such as Get and Post, are attribute modifiers. They add
// hx-get, hx-post, and related attributes that point to package-local fragment
// handlers served by Handler.
//
// Fragment handlers must be named top-level package functions. Dynamic
// functions such as closures and method values are rejected so the handler
// registry stays bounded. Rendering a template registers the handlers it
// references; load-balanced applications should preregister handlers during
// startup with MustRegister.
package hx
