package hx

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/doors-dev/gox"
	"github.com/doors-dev/goxx"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func TestHTMXE2E(t *testing.T) {
	resetGlobals()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		out, err := goxx.Render(r.Context(), testPage())
		if err != nil {
			http.Error(w, "render failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = out.WriteTo(w)
	})
	mux.HandleFunc(Prefix(), Handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	browserURL := launcher.New().NoSandbox(true).MustLaunch()
	browser := rod.New().ControlURL(browserURL).MustConnect()
	t.Cleanup(browser.MustClose)

	page := browser.MustPage(server.URL).MustWaitLoad()
	page.MustWait(`() => window.htmx !== undefined`)

	clickAndRequireText(t, page, "#get-button", "#get-value", "loaded by GET")
	clickAndRequireText(t, page, "#post-button", "#post-value", "created by POST")
	clickAndRequireText(t, page, "#put-button", "#put-value", "replaced by PUT")
	clickAndRequireText(t, page, "#patch-button", "#patch-value", "patched by PATCH")
	clickAndRequireText(t, page, "#delete-button", "#delete-value", "deleted by DELETE")
}

func TestPostModifierRegistersHandlerAndRendersAttribute(t *testing.T) {
	resetGlobals()

	got := renderString(t, testButton())

	id, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	want := `hx-post="` + Prefix() + id + `"`
	if !strings.Contains(got, want) {
		t.Fatalf("rendered html = %q, want it to contain %q", got, want)
	}
}

func TestHandlerDispatchesRegisteredFragment(t *testing.T) {
	resetGlobals()

	id, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, Prefix()+id, nil)
	Handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Body.String(); got != `<span>ok</span>` {
		t.Fatalf("body = %q, want fragment html", got)
	}
	if got := rec.Header().Get("X-HX-Test"); got != "yes" {
		t.Fatalf("X-HX-Test header = %q, want yes", got)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "seen" || cookies[0].Value != "true" {
		t.Fatalf("cookies = %#v, want seen=true", cookies)
	}
}

func TestHandlerAllowsAnyRequestMethod(t *testing.T) {
	resetGlobals()

	id, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, Prefix()+id, nil)
	Handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestRegisterStableIDDoesNotGrowHandlerRegistry(t *testing.T) {
	resetGlobals()

	first, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	second, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	if first != second {
		t.Fatalf("second id = %q, want same id %q", second, first)
	}
	if got := handlerCount(); got != 1 {
		t.Fatalf("handler count = %d, want 1", got)
	}
}

func TestModifiersUseHandlerIDIndependentOfMethod(t *testing.T) {
	resetGlobals()

	got := renderString(t, sameHandlerMethodButtons())
	id, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	for _, want := range []string{
		`hx-get="` + Prefix() + id + `"`,
		`hx-post="` + Prefix() + id + `"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered html = %q, want it to contain %q", got, want)
		}
	}
	if got := handlerCount(); got != 1 {
		t.Fatalf("handler count = %d, want 1", got)
	}
}

func TestRegisterRejectsDynamicHandlers(t *testing.T) {
	resetGlobals()

	_, err := Register(nil)
	requireErrorContains(t, err, "hx: handler must not be nil")
	_, err = Register(closureFragment("dynamic"))
	requireErrorContains(t, err, "closures and method values are not supported")
	var value methodFragment
	_, err = Register(value.fragment)
	requireErrorContains(t, err, "closures and method values are not supported")
	if got := handlerCount(); got != 0 {
		t.Fatalf("handler count = %d, want 0", got)
	}
}

func TestTopLevelFunctionNameDetection(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "main.fragment", want: true},
		{name: "github.com/example/my-app.fragment", want: true},
		{name: "github.com/example/hx.fragment", want: true},
		{name: "github.com/example/hx.fragment.func1", want: false},
		{name: "github.com/example/hx.init.func1", want: false},
		{name: "github.com/example/hx.methodFragment.fragment-fm", want: false},
		{name: "github.com/example/hx.(*methodFragment).fragment", want: false},
		{name: "github.com/example/hx.Fragment[go.shape.string]", want: false},
		{name: "github.com/example/hx.123fragment", want: false},
		{name: "", want: false},
	}

	for _, tt := range tests {
		if got := funRegexp.MatchString(tt.name); got != tt.want {
			t.Fatalf("function name match for %q = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestRepeatedRendersDoNotGrowHandlerRegistry(t *testing.T) {
	resetGlobals()

	for range 10 {
		_ = renderString(t, testPage())
	}
	if got, want := handlerCount(), 5; got != want {
		t.Fatalf("handler count after repeated page renders = %d, want %d", got, want)
	}
}

func TestDynamicModifierDoesNotRegisterHandler(t *testing.T) {
	resetGlobals()

	var b strings.Builder
	err := dynamicButton(closureFragment("dynamic")).Print(context.Background(), goxx.NewPrinter(&b))
	requireErrorContains(t, err, "hx: register hx-post handler")
	if got := handlerCount(); got != 0 {
		t.Fatalf("handler count = %d, want 0", got)
	}
}

func TestSettingsPrefix(t *testing.T) {
	resetGlobals()

	SetPrefix("/hx")
	if got := Prefix(); got != "/hx/" {
		t.Fatalf("Prefix() = %q, want /hx/", got)
	}
	assertPanicsContains(t, "is not URL path safe", func() {
		SetPrefix("not ok")
	})
	assertPanicsContains(t, "hx: prefix must not be empty", func() {
		SetPrefix("/")
	})

	got := renderString(t, testButton())
	id, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}
	want := `hx-post="/hx/` + id + `"`
	if !strings.Contains(got, want) {
		t.Fatalf("rendered html = %q, want it to contain %q", got, want)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, Prefix()+id, nil)
	Handler(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status with custom prefix = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestSettingsOptions(t *testing.T) {
	resetGlobals()
	SetOptions(goxx.WithPrinter(func(w io.Writer) gox.Printer {
		_, _ = io.WriteString(w, "<!--custom-printer-->")
		return gox.NewPrinter(w)
	}))

	id, err := Register(testFragment)
	if err != nil {
		t.Fatalf("Register() error = %v, want nil", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, Prefix()+id, nil)
	Handler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if got := rec.Body.String(); !strings.HasPrefix(got, "<!--custom-printer-->") {
		t.Fatalf("body = %q, want custom printer marker prefix", got)
	}
}

func renderString(t *testing.T, el gox.Elem) string {
	t.Helper()
	var b strings.Builder
	if err := el.Print(context.Background(), goxx.NewPrinter(&b)); err != nil {
		t.Fatalf("Print() error = %v, want nil", err)
	}
	return b.String()
}

func clickAndRequireText(t *testing.T, page *rod.Page, button, result, want string) {
	t.Helper()
	page.MustElement(button).MustClick()
	got := page.MustElement(result).MustText()
	if got != want {
		t.Fatalf("%s text = %q, want %q", result, got, want)
	}
}

func handlerCount() int {
	count := 0
	handlers.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

func closureFragment(label string) HandlerFunc {
	return func(_ Responser, _ *http.Request) gox.Elem {
		return textFragment(label)
	}
}

type methodFragment struct{}

func (methodFragment) fragment(_ Responser, _ *http.Request) gox.Elem {
	return textFragment("method")
}

func resetGlobals() {
	prefix = "/~/"
	options = nil
	handlers = sync.Map{}
}

func requireErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want it to contain %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func assertPanicsContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil {
			t.Fatal("function did not panic")
		}
		if !strings.Contains(got.(string), want) {
			t.Fatalf("panic = %q, want it to contain %q", got, want)
		}
	}()
	fn()
}
