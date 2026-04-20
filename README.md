# goxx
[![codecov](https://codecov.io/gh/doors-dev/goxx/branch/main/graph/badge.svg)](https://codecov.io/gh/doors-dev/goxx)
[![Go Report Card](https://goreportcard.com/badge/github.com/doors-dev/goxx)](https://goreportcard.com/report/github.com/doors-dev/goxx)
[![Go Reference](https://pkg.go.dev/badge/github.com/doors-dev/goxx.svg)](https://pkg.go.dev/github.com/doors-dev/goxx)

`goxx` is an extension package for [`github.com/doors-dev/gox`](https://github.com/doors-dev/gox).

GoX itself is intentionally minimal. `goxx` adds a few more specific,
but still common, rendering helpers on top:

- a parallel printer for independent slow template fragments
- composable class helpers
- small proxy utilities for building attribute-oriented helpers

Suggestions for extending this package are welcome. If you need another common
helper, please create an issue.

## Doors Compatibility

`goxx` is not fully compatible with [`github.com/doors-dev/doors`](https://github.com/doors-dev/doors).
When you are building a Doors app, prefer helpers from Doors itself or from the
GoX core package unless you know this package fits your rendering pipeline.

## Install

```sh
go get github.com/doors-dev/goxx
```

## Printer

### Parallel Rendering

Use `goxx.NewPrinter` instead of `gox.NewPrinter`, then mark independent
fragments with `~>(goxx.Parallel())`.

```go
func handlePage(w http.ResponseWriter, r *http.Request) {
    if err := Page().Print(r.Context(), goxx.NewPrinter(w)); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}
```

```go
elem Page() {
    <main>
        <h1>Dashboard</h1>

        ~>(goxx.Parallel()) <section>
            ~(SlowStats())
        </section>

        ~>(goxx.Parallel()) <aside>
            ~(SlowSidebar())
        </aside>
    </main>
}
```

Use `Parallel` for fragments that can wait independently, such as database
queries, external API calls, filesystem reads, or expensive calculations. Output
order stays the same as the template order, even if a later branch finishes
first.

By default, `NewPrinter` uses seven background workers plus the caller goroutine.

```go
// Use 16 background workers.
printer := goxx.NewPrinter(w, goxx.OptionWorkers(16))

// Use plain goroutines instead of a bounded worker pool.
printer = goxx.NewPrinter(w, goxx.OptionWorkers(0))
```

### Printer Extensions

`OptionPrinter` lets you add your own printer to the pipeline. It is a factory
because parallel rendering writes each branch to its own buffer.

```go
printer := goxx.NewPrinter(w, goxx.OptionPrinter(func(w io.Writer) gox.Printer {
    return MyPrinter(w)
}))
```

If your custom printer wants expanded content instead of `*gox.JobComp` values,
use `OptionFlat`.

```go
printer := goxx.NewPrinter(
    w,
    goxx.OptionFlat(),
    goxx.OptionPrinter(func(w io.Writer) gox.Printer {
        return MyPrinter(w)
    }),
)
```

### Error Handling

`NewPrinter` buffers rendered output before writing it to the final `io.Writer`.
If rendering fails before the final write, nothing is written to that writer.
This is useful in HTTP handlers because you can still choose the response status.

Use `WriterError` to detect errors from the final writer itself.

```go
err := Page().Print(r.Context(), goxx.NewPrinter(w))
if err == nil {
    return
}

if err, ok := goxx.WriterError(err); ok {
    slog.Warn("response write failed", "err", err)
    return
}

if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
    slog.Debug("render stopped before completion", "err", err)
    return
}

http.Error(w, "render failed", http.StatusInternalServerError)
```

## Class Helpers

`Class` builds immutable class modifiers. Inputs are split with `strings.Fields`,
so variadic and space-separated forms are equivalent.

```go
goxx.Class("button", "primary")
goxx.Class("button primary")
goxx.Class("button").Add("primary").Remove("hidden")
```

You can use `Classes` as an attribute modifier:

```go
elem Button() {
    <button (goxx.Class("button primary")) class="wide">Save</button>
}
```

Or as a class attribute value:

```go
elem Button() {
    <button class=(goxx.Class("button", "primary"))>Save</button>
}
```

Or as a proxy. The class modifier propagates through components and containers
until it reaches the first real element:

```go
elem Button() {
    ~>(goxx.Class("button primary")) <button>Save</button>
}
```

`Remove` is useful when wrapping a component that already has a class:

```go
elem BaseButton() {
    <button class="button disabled">Save</button>
}

elem EnabledButton() {
    ~>(goxx.Class("primary").Remove("disabled")) ~(BaseButton())
}
```

`Remove` filters matching classes no matter whether they were added before or
after the removal:

```go
goxx.Class("button hidden").Remove("hidden").String() // "button"
```

## Proxy Helpers

`ProxyMod` is useful when building helpers that attach an attribute modifier to
another element or component. It carries the modifier through leading components
or containers until the first real element, applies it once, and leaves later
siblings unchanged.

One practical use is adding test or integration attributes to components without
adding those attributes to every component API:

```go
func TestID(id string) gox.Proxy {
    return goxx.ProxyMod(gox.ModifyFunc(func(_ context.Context, _ string, attrs gox.Attrs) error {
        attrs.Get("data-testid").Set(id)
        return nil
    }))
}
```

```go
elem SaveButton() {
    <button class="button">Save</button>
}

elem Toolbar() {
    ~>(TestID("save-button")) ~(SaveButton())
}
```

`goxx.Class(...).Proxy(...)` is built on this behavior.
