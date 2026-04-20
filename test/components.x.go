// Managed by GoX v0.1.25

//line components.gox:1
package test

import (
	"context"
	"io"
	"time"
	
	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
)

// this should render ~500ms

func render(w io.Writer) error {
	p := gox.NewPrinter(w)
	return Parallel().Print(context.Background(), p)
}

//line components.gox:19
func Parallel() gox.Elem {
	return gox.Elem(func(__c gox.Cursor) (__e error) {
		ctx := __c.Context(); _ = ctx
//line components.gox:20
		__e = goxx.Parallel().Proxy(__c, gox.Elem(func(__c gox.Cursor) (__e error) {
			ctx := __c.Context(); _ = ctx
			__e = __c.InitContainer(); if __e != nil { return }
			{
//line components.gox:22
				<-time.After(time.Second / 2)

//line components.gox:24
				__e = goxx.Parallel().Proxy(__c, gox.Elem(func(__c gox.Cursor) (__e error) {
					ctx := __c.Context(); _ = ctx
					__e = __c.InitContainer(); if __e != nil { return }
					{
//line components.gox:26
						<-time.After(time.Second / 2)

//line components.gox:28
						__e = goxx.Parallel().Proxy(__c, gox.Elem(func(__c gox.Cursor) (__e error) {
							ctx := __c.Context(); _ = ctx
							__e = __c.InitContainer(); if __e != nil { return }
							{
//line components.gox:30
								<-time.After(time.Second / 2)

							}
							__e = __c.Close(); if __e != nil { return }
						return })); if __e != nil { return }
					}
					__e = __c.Close(); if __e != nil { return }
				return })); if __e != nil { return }
			}
			__e = __c.Close(); if __e != nil { return }
		return })); if __e != nil { return }
//line components.gox:35
		__e = goxx.Parallel().Proxy(__c, gox.Elem(func(__c gox.Cursor) (__e error) {
			ctx := __c.Context(); _ = ctx
			__e = __c.InitContainer(); if __e != nil { return }
			{
//line components.gox:37
				<-time.After(time.Second / 2)

			}
			__e = __c.Close(); if __e != nil { return }
		return })); if __e != nil { return }
//line components.gox:41
		<-time.After(time.Second / 2)

	return })
//line components.gox:43
}
