package goxx

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"unicode"

	"github.com/doors-dev/gox"
)

func Class(classes ...string) Classes {
	return Classes{}.Add(classes...)
}

type Classes struct {
	composed string
	stored   string
}

func (c Classes) Remove(classes ...string) Classes {
	builder := strings.Builder{}
	first := true
main:
	for class := range strings.FieldsSeq(c.composed) {
		for _, remove := range classes {
			for removeClass := range strings.FieldsSeq(remove) {
				if removeClass == class {
					continue main
				}
			}
		}
		if first {
			first = false
		} else {
			builder.WriteByte(' ')
		}
		builder.WriteString(class)
	}
	c.composed = builder.String()
	return c
}

func (c Classes) Add(classes ...string) Classes {
	builder := strings.Builder{}
	prev := string(c.composed)
	builder.WriteString(prev)
	for _, class := range classes {
		if !rightSpaced(prev) && !leftSpaced(class) {
			builder.WriteByte(' ')
		}
		builder.WriteString(class)
		prev = class
	}
	c.composed = builder.String()
	return c
}

func (c Classes) Join(classes ...Classes) Classes {
	builder := strings.Builder{}
	prev := c.composed
	builder.WriteString(prev)
	for _, c := range classes {
		if !rightSpaced(prev) && !leftSpaced(c.composed) {
			builder.WriteByte(' ')
		}
		builder.WriteString(c.composed)
		prev = c.composed
	}
	c.composed = builder.String()
	return c
}

func (c Classes) String() string {
	w := &strings.Builder{}
	if err := c.Output(w); err != nil {
		panic(err)
	}
	return w.String()
}

func (c Classes) Output(w io.Writer) error {
	if _, err := io.WriteString(w, c.composed); err != nil {
		return err
	}
	if !rightSpaced(c.composed) && !leftSpaced(c.stored) {
		if _, err := io.WriteString(w, " "); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(w, c.stored); err != nil {
		return err
	}
	return nil
}

func (c Classes) Modify(ctx context.Context, tag string, atts gox.Attrs) error {
	atts.Get("class").Set(c)
	return nil
}

func (c Classes) Mutate(name string, prev any) any {
	if name != "class" {
		slog.Warn("Class helper is assined to not `class` attribute: " + name)
	}
	if prevc, ok := prev.(Classes); ok {
		return prevc.Join(c)
	}
	if s, ok := prev.(string); ok {
		c.stored = s
		return c
	}
	return c
}

func (c Classes) Proxy(cur gox.Cursor, el gox.Elem) error {
	return ProxyMod(c).Proxy(cur, el)
}


func rightSpaced(s string) bool {
	if len(s) == 0 {
		return true
	}
	return unicode.IsSpace(rune(s[len(s)-1]))
}

func leftSpaced(s string) bool {
	if len(s) == 0 {
		return true
	}
	return unicode.IsSpace(rune(s[0]))
}



