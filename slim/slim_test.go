package slim

import (
	"bytes"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattn/go-slim"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".slim")
	require.NoError(t, engine.Load())

	// Partials
	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect := `<div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div>`
	result := trim(buf.String())
	require.Equal(t, expect, result)

	// Single
	buf.Reset()
	err = engine.Render(&buf, "errors/404", map[string]interface{}{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect = `<h1>Hello, World!</h1>`
	result = trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".slim")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!doctype html><html><head><title>Main</title></head><body><div><div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div></div></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".slim")

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "")
	require.NoError(t, err)

	expect := `<div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

// customMap stands in for binding types like fiber.Map: a named map type whose
// underlying type is map[string]interface{}.
type customMap map[string]interface{}

func Test_Layout_Binding(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/child.slim", []byte("h1 = Title\n"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.slim", []byte("div\n  p = Title\n  == embed\n"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".slim")
	require.NoError(t, engine.Load())

	expect := `<div><p>Hello, World!</p><div><h1>Hello, World!</h1></div></div>`

	// A binding that is not literally a map[string]interface{} has to reach
	// the layout with its values intact, not just the embedded body.
	custom := customMap{"Title": "Hello, World!"}
	var buf bytes.Buffer
	err = engine.Render(&buf, "child", custom, "layout")
	require.NoError(t, err)
	require.Equal(t, expect, trim(buf.String()))
	require.Equal(t, customMap{"Title": "Hello, World!"}, custom)

	// And rendering must not write the embedded body back into the binding the
	// caller handed in.
	binding := map[string]interface{}{"Title": "Hello, World!"}
	buf.Reset()
	err = engine.Render(&buf, "child", binding, "layout")
	require.NoError(t, err)
	require.Equal(t, expect, trim(buf.String()))
	require.Equal(t, map[string]interface{}{"Title": "Hello, World!"}, binding)
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".slim")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!doctype html><html><head><title>Main</title></head><body><div><div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div></div></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".slim")
	engine.Reload(true) // Optional. Default: false
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/ShouldReload.slim", []byte("p after ShouldReload\n"), 0o600)
	require.NoError(t, err)

	defer func() {
		err := os.WriteFile("./views/ShouldReload.slim", []byte("p before ShouldReload\n"), 0o600)
		require.NoError(t, err)
	}()

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "ShouldReload", nil)
	require.NoError(t, err)

	expect := "<p>after ShouldReload</p>"
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_AddFuncMap(t *testing.T) {
	t.Parallel()
	// Create a temporary directory
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	// Create a temporary template file.
	err = os.WriteFile(dir+"/func_map.slim", []byte(`
h2 = lower(Var1)
p = upper(Var2)`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".slim")
	fm := map[string]interface{}{
		"lower": func(s ...slim.Value) (slim.Value, error) {
			slimvalue, ok := s[0].(string)
			require.True(t, ok)

			return strings.ToLower(slimvalue), nil
		},
		"upper": func(s ...slim.Value) (slim.Value, error) {
			slimvalue, ok := s[0].(string)
			require.True(t, ok)

			return strings.ToUpper(slimvalue), nil
		},
	}
	engine.AddFuncMap(fm)

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "func_map", map[string]interface{}{
		"Var1": "LOwEr",
		"Var2": "upPEr",
	})
	require.NoError(t, err)

	expect := `<h2>lower</h2><p>UPPER</p>`
	result := trim(buf.String())
	require.Equal(t, expect, result)

	// FuncMap
	fm2 := engine.FuncMap()
	_, ok := fm2["lower"]
	require.True(t, ok)
	_, ok = fm2["upper"]
	require.True(t, ok)
}

func Benchmark_Slim(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	engine := New("./views", ".slim")
	engine.AddFunc("isAdmin", func(s ...slim.Value) (slim.Value, error) {
		return s[0].(string) == "admin", nil
	})
	require.NoError(b, engine.Load())

	b.Run("simple", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			var buf bytes.Buffer
			//nolint:gosec,errcheck // Return value not needed for benchmark
			_ = engine.Render(&buf, "simple", map[string]interface{}{
				"Title": "Hello, World!",
			})
		}
	})

	b.Run("simple_asserted", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			var buf bytes.Buffer
			err := engine.Render(&buf, "simple", map[string]interface{}{
				"Title": "Hello, World!",
			})
			require.NoError(b, err)
			require.Equal(b, expectSimple, trim(buf.String()))
		}
	})
}

func Benchmark_Slim_Parallel(b *testing.B) {
	expectSimple := `<h1>Hello, Parallel!</h1>`
	engine := New("./views", ".slim")
	engine.AddFunc("isAdmin", func(s ...slim.Value) (slim.Value, error) {
		return s[0].(string) == "admin", nil
	})
	require.NoError(b, engine.Load())

	b.Run("simple", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		bb.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var buf bytes.Buffer
				//nolint:gosec,errcheck // Return value not needed for benchmark
				_ = engine.Render(&buf, "simple", map[string]interface{}{
					"Title": "Hello, Parallel!",
				})
			}
		})
	})

	b.Run("simple_asserted", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		bb.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var buf bytes.Buffer
				err := engine.Render(&buf, "simple", map[string]interface{}{
					"Title": "Hello, Parallel!",
				})
				require.NoError(bb, err)
				require.Equal(bb, expectSimple, trim(buf.String()))
			}
		})
	})
}

func Test_Load_Retry(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	views := dir + "/views"
	engine := New(views, ".slim")
	require.Error(t, engine.Load())

	// A failed load must not stick: once the cause is gone, the next render
	// has to reload and serve - Load stays failed-state aware on its own.
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(views+"/index.slim", []byte(`p ok`), 0o600))

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "index", nil))
	require.Equal(t, "<p>ok</p>", trim(buf.String()))
}
