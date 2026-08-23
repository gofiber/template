package template_test

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	template "github.com/gofiber/template/v2"
)

type customMap map[string]interface{}

func newEngine() *template.Engine {
	return &template.Engine{Funcmap: make(map[string]interface{})}
}

func Test_PreRenderCheck(t *testing.T) {
	t.Parallel()

	t.Run("not loaded", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		require.True(t, e.PreRenderCheck())
		require.False(t, e.Loaded)
	})

	t.Run("loaded", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		e.Loaded = true
		require.False(t, e.PreRenderCheck())
		require.True(t, e.Loaded)
	})

	t.Run("loaded with reload", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		e.Loaded = true
		e.Reload(true)
		require.True(t, e.PreRenderCheck())
		require.False(t, e.Loaded, "reloading has to clear the loaded flag")
	})

	t.Run("reload lifecycle", func(t *testing.T) {
		t.Parallel()
		e := newEngine()

		// Unloaded engine: keeps asking to be loaded.
		require.True(t, e.PreRenderCheck())
		e.Loaded = true

		// Loaded engine: answers false from here on, cached or not.
		require.False(t, e.PreRenderCheck())
		require.False(t, e.PreRenderCheck())

		// Turning reloading on has to take effect even though the previous
		// answer was cached, and has to stay in effect on every render.
		e.Reload(true)
		for range 3 {
			require.True(t, e.PreRenderCheck())
			require.False(t, e.Loaded)
			e.Loaded = true // as an engine's Load would
		}

		// And turning it back off has to settle again.
		e.Reload(false)
		require.False(t, e.PreRenderCheck())
		require.False(t, e.PreRenderCheck())
		require.True(t, e.Loaded)
	})

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		e.Loaded = true
		done := make(chan bool, 8)
		for range cap(done) {
			go func() { done <- e.PreRenderCheck() }()
		}
		for range cap(done) {
			require.False(t, <-done)
		}
	})
}

func Test_AcquireViewContext(t *testing.T) {
	t.Parallel()

	native := map[string]interface{}{"key": "value"}
	require.Equal(t, native, template.AcquireViewContext(native))

	custom := customMap{"key": "value"}
	require.Equal(t, map[string]interface{}{"key": "value"}, template.AcquireViewContext(custom))

	require.Equal(t, map[string]interface{}{"key": "value"}, template.AcquireViewContext(&custom))

	for name, binding := range map[string]interface{}{
		"nil":              nil,
		"nil pointer":      (*customMap)(nil),
		"nil map":          customMap(nil),
		"not a map":        "value",
		"non string keys":  map[int]string{1: "value"},
		"pointer to value": func() interface{} { v := "value"; return &v }(),
	} {
		require.Empty(t, template.AcquireViewContext(binding), name)
		require.NotNil(t, template.AcquireViewContext(binding), name)
	}
}

func Test_ViewContext_Range(t *testing.T) {
	t.Parallel()

	for name, binding := range map[string]interface{}{
		"native map": map[string]interface{}{"a": 1, "b": 2},
		"custom map": customMap{"a": 1, "b": 2},
		"pointer":    &customMap{"a": 1, "b": 2},
		"nil":        nil,
		"not a map":  "value",
		"empty":      customMap{},
	} {
		want := template.AcquireViewContext(binding)

		got := make(map[string]interface{}, template.ViewContextLen(binding))
		template.RangeViewContext(binding, func(key string, value interface{}) {
			got[key] = value
		})

		require.Equal(t, want, got, name)
		require.Len(t, want, template.ViewContextLen(binding), name)
	}
}

func Test_NewViewContext(t *testing.T) {
	t.Parallel()

	for name, binding := range map[string]interface{}{
		"native":    map[string]interface{}{"a": 1, "b": 2},
		"named":     customMap{"a": 1, "b": 2},
		"pointer":   &customMap{"a": 1, "b": 2},
		"nil":       nil,
		"not a map": "value",
	} {
		want := template.AcquireViewContext(binding)
		got := template.NewViewContext(binding, 1)
		require.Equal(t, want, got, name)

		// The result is always the caller's to write into: unlike
		// AcquireViewContext it must never be the binding itself.
		got["embed"] = "body"
		require.Equal(t, want, template.AcquireViewContext(binding), name)
	}
}

func Test_HasExtension(t *testing.T) {
	t.Parallel()

	require.True(t, template.HasExtension("views/index.html", ".html"))
	require.True(t, template.HasExtension("views/index.html.jet", ".html.jet"))
	require.True(t, template.HasExtension("a.html", ".html"))

	require.False(t, template.HasExtension(".html", ".html"), "a bare extension is not a template")
	require.False(t, template.HasExtension("views/index.htm", ".html"))
	require.False(t, template.HasExtension("views/index.html", ".jet"))
	require.False(t, template.HasExtension("", ".html"))
}

func Test_TemplateName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		directory string
		path      string
		extension string
		expect    string
	}{
		{"views", filepath.Join("views", "index.html"), ".html", "index"},
		{"./views", filepath.Join("views", "index.html"), ".html", "index"},
		{"views", filepath.Join("views", "partials", "footer.html"), ".html", "partials/footer"},
		{"/", "/index.html", ".html", "index"},
		{"views", filepath.Join("views", "index.html"), "", "index.html"},
		{"views", filepath.Join("views", "a.html.b.html"), ".html", "a.html.b"},
	} {
		name, err := template.TemplateName(tc.directory, tc.path, tc.extension)
		require.NoError(t, err)
		require.Equal(t, tc.expect, name)

	}
}

func Test_TemplateName_Error(t *testing.T) {
	t.Parallel()

	_, err := template.TemplateName("relative", string(filepath.Separator)+"absolute/index.html", ".html")
	require.Error(t, err)
}

func Test_UnsafeConversions(t *testing.T) {
	t.Parallel()

	require.Equal(t, "hello", template.UnsafeString([]byte("hello")))
	require.Empty(t, template.UnsafeString(nil))

	require.Equal(t, []byte("hello"), template.UnsafeBytes("hello"))
	require.Empty(t, template.UnsafeBytes(""))
}

func Test_ReadFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	require.NoError(t, os.WriteFile(path, []byte("<h1>Hello</h1>"), 0o600))

	buf, err := template.ReadFile(path, nil)
	require.NoError(t, err)
	require.Equal(t, "<h1>Hello</h1>", string(buf))

	buf, err = template.ReadFile("index.html", http.Dir(dir))
	require.NoError(t, err)
	require.Equal(t, "<h1>Hello</h1>", string(buf))

	_, err = template.ReadFile(filepath.Join(dir, "missing.html"), nil)
	require.Error(t, err)
}

func Test_Walk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "partials"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "partials", "footer.html"), []byte("footer"), 0o600))

	var names []string
	err := template.Walk(http.Dir(dir), "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() || !template.HasExtension(path, ".html") {
			return nil
		}
		name, err := template.TemplateName("/", path, ".html")
		if err != nil {
			return err
		}
		names = append(names, name)
		return nil
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"index", "partials/footer"}, names)

	err = template.Walk(http.Dir(dir), "/", func(_ string, _ os.FileInfo, _ error) error {
		return errors.New("boom")
	})
	require.Error(t, err)
}

func Benchmark_AcquireViewContext(b *testing.B) {
	// A native map is handed straight back; a named one - fiber.Map and
	// friends, which is what bindings usually are - is copied with plain Go;
	// anything else has to go through reflect.
	for name, binding := range map[string]interface{}{
		"native": map[string]interface{}{"Title": "Hello", "User": "admin"},
		"named":  customMap{"Title": "Hello", "User": "admin"},
		"typed":  map[string]string{"Title": "Hello", "User": "admin"},
	} {
		b.Run(name, func(bb *testing.B) {
			bb.ReportAllocs()
			for bb.Loop() {
				template.AcquireViewContext(binding)
			}
		})
	}
}
