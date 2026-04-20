package goxx_test

import (
	"strings"
	"testing"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
)

func TestProxyModAddsClassToFirstElement(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Init("span"); err != nil {
				return err
			}
			if err := cur.Submit(); err != nil {
				return err
			}
			if err := cur.Text("x"); err != nil {
				return err
			}
			return cur.Close()
		}))
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	const want = `<span class="hot">x</span>`
	if got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func TestProxyModCarriesClassThroughComponent(t *testing.T) {
	child := gox.Elem(func(cur gox.Cursor) error {
		if err := cur.Init("span"); err != nil {
			return err
		}
		if err := cur.AttrSet("class", "base"); err != nil {
			return err
		}
		if err := cur.Submit(); err != nil {
			return err
		}
		if err := cur.Text("x"); err != nil {
			return err
		}
		return cur.Close()
	})
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			return cur.Comp(child)
		}))
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	const want = `<span class="base hot">x</span>`
	if got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func TestProxyModNilModifierPassesThrough(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.ProxyMod(nil).Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := cur.Text("before|"); err != nil {
				return err
			}
			if err := spanWithClass("base")(cur); err != nil {
				return err
			}
			return cur.Text("|after")
		}))
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	const want = `before|<span class="base"></span>|after`
	if got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func TestProxyModLooksThroughContainer(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := cur.InitContainer(); err != nil {
				return err
			}
			if err := spanWithClass("base")(cur); err != nil {
				return err
			}
			if err := cur.Text("|after"); err != nil {
				return err
			}
			return cur.Close()
		}))
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	const want = `<span class="base hot"></span>|after`
	if got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func TestProxyModAppliesOnlyToFirstElement(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := emptySpan()(cur); err != nil {
				return err
			}
			return emptySpan()(cur)
		}))
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	const want = `<span class="hot"></span><span></span>`
	if got != want {
		t.Fatalf("Print() html = %q, want %q", got, want)
	}
}

func TestProxyModConsumesNilComponent(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			return cur.Comp(nilComp{})
		}))
	})

	got, err := renderString(root)
	if err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("Print() html = %q, want empty", got)
	}
}

func TestProxyModPreservesParallelMarker(t *testing.T) {
	forEachWorkerVariant(t, func(t *testing.T, variant workerVariant) {
		root := gox.Elem(func(cur gox.Cursor) error {
			return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
				return goxx.Parallel().Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
					if err := cur.Init("span"); err != nil {
						return err
					}
					if err := cur.Submit(); err != nil {
						return err
					}
					if err := cur.Text("x"); err != nil {
						return err
					}
					return cur.Close()
				}))
			}))
		})

		got, err := renderString(root, variant.opts...)
		if err != nil {
			t.Fatalf("Print() error = %v, want nil", err)
		}
		const want = `<span class="hot">x</span>`
		if got != want {
			t.Fatalf("Print() html = %q, want %q", got, want)
		}
	})
}

func TestProxyModRejectsTextFirstElement(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			return cur.Text("x")
		}))
	})

	_, err := renderString(root)
	if err == nil {
		t.Fatal("Print() error = nil, want error")
	}
	const want = "cannot attach an attribute modifier"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Print() error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestProxyModRejectsTextBeforeElementInContainer(t *testing.T) {
	root := gox.Elem(func(cur gox.Cursor) error {
		return goxx.Class("hot").Proxy(cur, gox.Elem(func(cur gox.Cursor) error {
			if err := cur.InitContainer(); err != nil {
				return err
			}
			if err := cur.Text("before"); err != nil {
				return err
			}
			return emptySpan()(cur)
		}))
	})

	_, err := renderString(root)
	if err == nil {
		t.Fatal("Print() error = nil, want error")
	}
	const want = "cannot attach an attribute modifier"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("Print() error = %q, want it to contain %q", err.Error(), want)
	}
}
