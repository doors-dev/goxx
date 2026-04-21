package goxx_test

import (
	"testing"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
)

func TestClassesBuildsFromSplitAndVariadicInputs(t *testing.T) {
	tests := []struct {
		name string
		in   goxx.Classes
		want string
	}{
		{
			name: "single",
			in:   goxx.Class("test"),
			want: "test",
		},
		{
			name: "variadic",
			in:   goxx.Class("test", "test1"),
			want: "test test1",
		},
		{
			name: "space separated",
			in:   goxx.Class("test test1"),
			want: "test test1",
		},
		{
			name: "trim and collapse spaces",
			in:   goxx.Class(" test  test1\t test2\n"),
			want: "test test1 test2",
		},
		{
			name: "add variadic",
			in:   goxx.Class("test").Add("test1", "test2"),
			want: "test test1 test2",
		},
		{
			name: "add space separated",
			in:   goxx.Class("test").Add("test1 test2"),
			want: "test test1 test2",
		},
		{
			name: "remove single",
			in:   goxx.Class("test test1 test2").Remove("test1"),
			want: "test test2",
		},
		{
			name: "remove variadic and space separated",
			in:   goxx.Class("test test1 test2 test3").Remove("test1", "test2 test3"),
			want: "test",
		},
		{
			name: "remove does not filter later adds",
			in:   goxx.Class("test test1").Remove("test1").Add("test1"),
			want: "test test1",
		},
		{
			name: "filter single",
			in:   goxx.Class("test test1 test2").Filter("test1"),
			want: "test test2",
		},
		{
			name: "filter variadic and space separated",
			in:   goxx.Class("test test1 test2 test3").Filter("test1", "test2 test3"),
			want: "test",
		},
		{
			name: "join keeps add and filter lists",
			in:   goxx.Class("test test1").Join(goxx.Class("test2").Filter("test1")),
			want: "test test2",
		},
		{
			name: "filter can be added before target class",
			in:   goxx.Class("test").Filter("test1").Add("test1 test2"),
			want: "test test2",
		},
		{
			name: "remove does not undo filters",
			in:   goxx.Class("test").Filter("test1").Remove("test1").Add("test1"),
			want: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassesMethodsAreImmutable(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		base := goxx.Class("base0 base1 base2")
		left := base.Add("left")
		right := base.Add("right")

		requireClassString(t, base, "base0 base1 base2")
		requireClassString(t, left, "base0 base1 base2 left")
		requireClassString(t, right, "base0 base1 base2 right")
	})

	t.Run("Filter", func(t *testing.T) {
		base := goxx.Class("keep left right").Filter("unused0 unused1 unused2")
		left := base.Filter("left")
		right := base.Filter("right")

		requireClassString(t, base, "keep left right")
		requireClassString(t, left, "keep right")
		requireClassString(t, right, "keep left")
	})

	t.Run("Remove", func(t *testing.T) {
		base := goxx.Class("keep left right")
		left := base.Remove("left")
		right := base.Remove("right")

		requireClassString(t, base, "keep left right")
		requireClassString(t, left, "keep right")
		requireClassString(t, right, "keep left")
	})

	t.Run("Join", func(t *testing.T) {
		base := goxx.Class("base0 base1 base2")
		left := base.Join(goxx.Class("left"))
		right := base.Join(goxx.Class("right"))

		requireClassString(t, base, "base0 base1 base2")
		requireClassString(t, left, "base0 base1 base2 left")
		requireClassString(t, right, "base0 base1 base2 right")
	})

	t.Run("Clone", func(t *testing.T) {
		base := goxx.Class("base0 base1 base2").Filter("unused0 unused1 unused2")
		clone := base.Clone()
		left := clone.Add("left").Remove("base1")
		right := clone.Add("right").Filter("base2")

		requireClassString(t, base, "base0 base1 base2")
		requireClassString(t, clone, "base0 base1 base2")
		requireClassString(t, left, "base0 base2 left")
		requireClassString(t, right, "base0 base1 right")
	})
}

func TestClassesRenderThroughGeneratedSyntaxShapes(t *testing.T) {
	test := gox.Elem(func(cur gox.Cursor) error {
		if err := cur.Init("span"); err != nil {
			return err
		}
		if err := cur.Set("class", "test2"); err != nil {
			return err
		}
		if err := cur.Submit(); err != nil {
			return err
		}
		return cur.Close()
	})
	test1 := gox.Elem(func(cur gox.Cursor) error {
		return cur.Any(test)
	})
	test2 := gox.Elem(func(cur gox.Cursor) error {
		if err := cur.InitContainer(); err != nil {
			return err
		}
		if err := cur.Any(test); err != nil {
			return err
		}
		return cur.Close()
	})

	root := gox.Elem(func(cur gox.Cursor) error {
		if err := spanWithClassMod(cur, goxx.Class("test"), ""); err != nil {
			return err
		}
		if err := spanWithClassMod(cur, goxx.Class("test").Filter("test2"), "test2"); err != nil {
			return err
		}
		if err := spanWithClassAttr(cur, goxx.Class("test")); err != nil {
			return err
		}
		if err := goxx.Class("test").Proxy(cur, emptySpan()); err != nil {
			return err
		}
		if err := goxx.Class("test").Filter("test2").Proxy(cur, spanWithClass("test2")); err != nil {
			return err
		}
		if err := goxx.Class("test").Proxy(cur, test); err != nil {
			return err
		}
		if err := goxx.Class("test").Filter("test2").Proxy(cur, test); err != nil {
			return err
		}
		if err := goxx.Class("test").Proxy(cur, test1); err != nil {
			return err
		}
		if err := goxx.Class("test").Filter("test2").Proxy(cur, test1); err != nil {
			return err
		}
		if err := goxx.Class("test").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := cur.InitContainer(); err != nil {
				return err
			}
			if err := emptySpan()(cur); err != nil {
				return err
			}
			return cur.Close()
		})); err != nil {
			return err
		}
		if err := goxx.Class("test").Filter("test2").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := cur.InitContainer(); err != nil {
				return err
			}
			if err := spanWithClass("test2")(cur); err != nil {
				return err
			}
			return cur.Close()
		})); err != nil {
			return err
		}
		if err := goxx.Class("test").Proxy(cur, test2); err != nil {
			return err
		}
		return goxx.Class("test").Filter("test2").Proxy(cur, test2)
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	want := stringsJoin(
		`<span class="test"></span>`,
		`<span class="test"></span>`,
		`<span class="test"></span>`,
		`<span class="test"></span>`,
		`<span class="test"></span>`,
		`<span class="test2 test"></span>`,
		`<span class="test"></span>`,
		`<span class="test2 test"></span>`,
		`<span class="test"></span>`,
		`<span class="test"></span>`,
		`<span class="test"></span>`,
		`<span class="test2 test"></span>`,
		`<span class="test"></span>`,
	)
	if got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func requireClassString(t *testing.T, classes goxx.Classes, want string) {
	t.Helper()
	if got := classes.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func spanWithClassMod(cur gox.Cursor, classes goxx.Classes, class string) error {
	if err := cur.Init("span"); err != nil {
		return err
	}
	if err := cur.Modify(classes); err != nil {
		return err
	}
	if class != "" {
		if err := cur.Set("class", class); err != nil {
			return err
		}
	}
	if err := cur.Submit(); err != nil {
		return err
	}
	return cur.Close()
}

func spanWithClassAttr(cur gox.Cursor, classes goxx.Classes) error {
	if err := cur.Init("span"); err != nil {
		return err
	}
	if err := cur.Set("class", classes); err != nil {
		return err
	}
	if err := cur.Submit(); err != nil {
		return err
	}
	return cur.Close()
}

func emptySpan() gox.Elem {
	return spanWithClass("")
}

func spanWithClass(class string) gox.Elem {
	return gox.Elem(func(cur gox.Cursor) error {
		if err := cur.Init("span"); err != nil {
			return err
		}
		if class != "" {
			if err := cur.Set("class", class); err != nil {
				return err
			}
		}
		if err := cur.Submit(); err != nil {
			return err
		}
		return cur.Close()
	})
}

func stringsJoin(parts ...string) string {
	var out string
	for _, part := range parts {
		out += part
	}
	return out
}
