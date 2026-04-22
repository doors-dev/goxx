// Managed by GoX v0.1.28

//line handler_test.gox:1
package hx

import (
	"net/http"
	
	"github.com/doors-dev/gox"
)

//line handler_test.gox:9
func testButton() gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("button"); if __e != nil { return }
		{
//line handler_test.gox:10
			__e = __c.Modify(Post(testFragment)); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("Post!"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:11
}

//line handler_test.gox:13
func sameHandlerMethodButtons() gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("button"); if __e != nil { return }
		{
//line handler_test.gox:14
			__e = __c.Modify(Get(testFragment)); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("Get"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
		__e = __c.Init("button"); if __e != nil { return }
		{
//line handler_test.gox:15
			__e = __c.Modify(Post(testFragment)); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("Post"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:16
}

//line handler_test.gox:18
func dynamicButton(h HandlerFunc) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("button"); if __e != nil { return }
		{
//line handler_test.gox:19
			__e = __c.Modify(Post(h)); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("Dynamic"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:20
}

//line handler_test.gox:22
func testPage() gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("script"); if __e != nil { return }
		{
//line handler_test.gox:23
			__e = __c.Set("src", "https://unpkg.com/htmx.org@1.9.11/dist/htmx.min.js"); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Raw(""); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
		__e = __c.Init("main"); if __e != nil { return }
		{
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Init("section"); if __e != nil { return }
			{
				__e = __c.Submit(); if __e != nil { return }
				__e = __c.Init("div"); if __e != nil { return }
				{
//line handler_test.gox:26
					__e = __c.Set("id", "get-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
				__e = __c.Init("button"); if __e != nil { return }
				{
//line handler_test.gox:27
					__e = __c.Set("id", "get-button"); if __e != nil { return }
//line handler_test.gox:27
					__e = __c.Modify(Get(getFragment)); if __e != nil { return }
//line handler_test.gox:27
					__e = __c.Set("hx-target", "#get-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
					__e = __c.Text("Load"); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
			}
			__e = __c.Close(); if __e != nil { return }
			__e = __c.Init("section"); if __e != nil { return }
			{
				__e = __c.Submit(); if __e != nil { return }
				__e = __c.Init("div"); if __e != nil { return }
				{
//line handler_test.gox:30
					__e = __c.Set("id", "post-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
				__e = __c.Init("button"); if __e != nil { return }
				{
//line handler_test.gox:31
					__e = __c.Set("id", "post-button"); if __e != nil { return }
//line handler_test.gox:31
					__e = __c.Modify(Post(postFragment)); if __e != nil { return }
//line handler_test.gox:31
					__e = __c.Set("hx-target", "#post-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
					__e = __c.Text("Create"); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
			}
			__e = __c.Close(); if __e != nil { return }
			__e = __c.Init("section"); if __e != nil { return }
			{
				__e = __c.Submit(); if __e != nil { return }
				__e = __c.Init("div"); if __e != nil { return }
				{
//line handler_test.gox:34
					__e = __c.Set("id", "put-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
				__e = __c.Init("button"); if __e != nil { return }
				{
//line handler_test.gox:35
					__e = __c.Set("id", "put-button"); if __e != nil { return }
//line handler_test.gox:35
					__e = __c.Modify(Put(putFragment)); if __e != nil { return }
//line handler_test.gox:35
					__e = __c.Set("hx-target", "#put-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
					__e = __c.Text("Replace"); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
			}
			__e = __c.Close(); if __e != nil { return }
			__e = __c.Init("section"); if __e != nil { return }
			{
				__e = __c.Submit(); if __e != nil { return }
				__e = __c.Init("div"); if __e != nil { return }
				{
//line handler_test.gox:38
					__e = __c.Set("id", "patch-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
				__e = __c.Init("button"); if __e != nil { return }
				{
//line handler_test.gox:39
					__e = __c.Set("id", "patch-button"); if __e != nil { return }
//line handler_test.gox:39
					__e = __c.Modify(Patch(patchFragment)); if __e != nil { return }
//line handler_test.gox:39
					__e = __c.Set("hx-target", "#patch-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
					__e = __c.Text("Patch"); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
			}
			__e = __c.Close(); if __e != nil { return }
			__e = __c.Init("section"); if __e != nil { return }
			{
				__e = __c.Submit(); if __e != nil { return }
				__e = __c.Init("div"); if __e != nil { return }
				{
//line handler_test.gox:42
					__e = __c.Set("id", "delete-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
					__e = __c.Init("span"); if __e != nil { return }
					{
//line handler_test.gox:42
						__e = __c.Set("id", "delete-ready"); if __e != nil { return }
						__e = __c.Submit(); if __e != nil { return }
						__e = __c.Text("ready"); if __e != nil { return }
					}
					__e = __c.Close(); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
				__e = __c.Init("button"); if __e != nil { return }
				{
//line handler_test.gox:43
					__e = __c.Set("id", "delete-button"); if __e != nil { return }
//line handler_test.gox:43
					__e = __c.Modify(Delete(deleteFragment)); if __e != nil { return }
//line handler_test.gox:43
					__e = __c.Set("hx-target", "#delete-result"); if __e != nil { return }
					__e = __c.Submit(); if __e != nil { return }
					__e = __c.Text("Delete"); if __e != nil { return }
				}
				__e = __c.Close(); if __e != nil { return }
			}
			__e = __c.Close(); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:46
}

//line handler_test.gox:48
func getFragment(_ Responser, _ *http.Request) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("span"); if __e != nil { return }
		{
//line handler_test.gox:49
			__e = __c.Set("id", "get-value"); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("loaded by GET"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:50
}

//line handler_test.gox:52
func postFragment(w Responser, _ *http.Request) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
//line handler_test.gox:54
		w.SetStatus(http.StatusCreated)

		__e = __c.Init("span"); if __e != nil { return }
		{
//line handler_test.gox:56
			__e = __c.Set("id", "post-value"); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("created by POST"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:57
}

//line handler_test.gox:59
func putFragment(_ Responser, _ *http.Request) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("span"); if __e != nil { return }
		{
//line handler_test.gox:60
			__e = __c.Set("id", "put-value"); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("replaced by PUT"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:61
}

//line handler_test.gox:63
func patchFragment(_ Responser, _ *http.Request) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("span"); if __e != nil { return }
		{
//line handler_test.gox:64
			__e = __c.Set("id", "patch-value"); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("patched by PATCH"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:65
}

//line handler_test.gox:67
func deleteFragment(_ Responser, _ *http.Request) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
		__e = __c.Init("span"); if __e != nil { return }
		{
//line handler_test.gox:68
			__e = __c.Set("id", "delete-value"); if __e != nil { return }
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("deleted by DELETE"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:69
}

//line handler_test.gox:71
func textFragment(label string) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
//line handler_test.gox:72
		__e = __c.Any(label); if __e != nil { return }
	return })
//line handler_test.gox:73
}

//line handler_test.gox:75
func testFragment(w Responser, _ *http.Request) gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
//line handler_test.gox:77
		w.Header().Set("X-HX-Test", "yes")
		w.SetCookie(&http.Cookie{Name: "seen", Value: "true"})
		w.SetStatus(http.StatusCreated)

		__e = __c.Init("span"); if __e != nil { return }
		{
			__e = __c.Submit(); if __e != nil { return }
			__e = __c.Text("ok"); if __e != nil { return }
		}
		__e = __c.Close(); if __e != nil { return }
	return })
//line handler_test.gox:82
}
