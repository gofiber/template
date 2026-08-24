package jet

import (
	"bytes"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/CloudyKit/jet/v6"
	"github.com/stretchr/testify/require"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".jet")
	require.NoError(t, engine.Load())
	// Partials
	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
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
	engine := New("./views", ".jet")

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Title</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Layout_Error(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".jet")

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		// "Title": "Hello, World!",
	}, "layouts/main")
	require.NotNil(t, err)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".jet")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "")
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_VarMap_Isolation(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/page.jet", []byte(`{{ Title = "MUTATED" }}<p>{{ Title }}</p>`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".jet")
	require.NoError(t, engine.Load())

	// jet assigns back into the VarMap it is handed, so the engine has to render
	// from one of its own - otherwise the caller's map changes under them, and
	// concurrent renders sharing it race.
	shared := jet.VarMap{}
	shared.Set("Title", "original")

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "page", shared))
	require.Equal(t, "original", shared["Title"].String())
	before := buf.String()

	// A regression that only shows up under contention would leave the shared map
	// intact but the renders themselves failing, so keep each one's result.
	const rounds = 20
	errs := make([]error, rounds)
	outs := make([]string, rounds)
	var wg sync.WaitGroup
	for i := range rounds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			errs[i] = engine.Render(&buf, "page", shared)
			outs[i] = buf.String()
		}()
	}
	wg.Wait()
	require.Equal(t, "original", shared["Title"].String())

	for i := range rounds {
		require.NoError(t, errs[i])
		require.Equal(t, before, outs[i])
	}
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".jet")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Title</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".jet")
	engine.Reload(true) // Optional. Default: false
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/ShouldReload.jet", []byte("after ShouldReload\n"), 0o600)
	require.NoError(t, err)
	defer func() {
		err := os.WriteFile("./views/ShouldReload.jet", []byte("before ShouldReload\n"), 0o600)
		require.NoError(t, err)
	}()

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "ShouldReload", nil)
	require.NoError(t, err)

	expect := "after ShouldReload"
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
	err = os.WriteFile(dir+"/func_map.jet", []byte(`<h2>{{Var1|lower}}</h2><p>{{Var2|upper}}</p>`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".jet")
	fm := map[string]interface{}{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
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

func Benchmark_Jet(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Title</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".jet")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	require.NoError(b, engine.Load())

	b.Run("simple", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		var buf bytes.Buffer
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			//nolint:gosec,errcheck // Return value not needed for benchmark
			_ = engine.Render(&buf, "simple", map[string]interface{}{
				"Title": "Hello, World!",
			})
		}
	})

	b.Run("extended", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		var buf bytes.Buffer
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			//nolint:gosec,errcheck // Return value not needed for benchmark
			_ = engine.Render(&buf, "extended", map[string]interface{}{
				"User": "admin",
			}, "layouts/main")
		}
	})

	b.Run("simple_asserted", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		var buf bytes.Buffer
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err := engine.Render(&buf, "simple", map[string]interface{}{
				"Title": "Hello, World!",
			})
			require.NoError(bb, err)
			require.Equal(bb, expectSimple, trim(buf.String()))
		}
	})

	b.Run("extended_asserted", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		var buf bytes.Buffer
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err := engine.Render(&buf, "extended", map[string]interface{}{
				"User": "admin",
			}, "layouts/main")
			require.NoError(bb, err)
			require.Equal(bb, expectExtended, trim(buf.String()))
		}
	})
}

func Benchmark_Jet_Parallel(b *testing.B) {
	expectSimple := `<h1>Hello, Parallel!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Title</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".jet")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
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

	b.Run("extended", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		bb.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var buf bytes.Buffer
				//nolint:gosec,errcheck // Return value not needed for benchmark
				_ = engine.Render(&buf, "extended", map[string]interface{}{
					"User": "admin",
				}, "layouts/main")
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

	b.Run("extended_asserted", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		bb.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var buf bytes.Buffer
				err := engine.Render(&buf, "extended", map[string]interface{}{
					"User": "admin",
				}, "layouts/main")
				require.NoError(bb, err)
				require.Equal(bb, expectExtended, trim(buf.String()))
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
	engine := New(views, ".jet")
	require.Error(t, engine.Load())

	// A failed load must not stick: once the cause is gone, the next load
	// parses and renders.
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(views+"/index.jet", []byte(`OK-{{ Title }}`), 0o600))
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "index", map[string]interface{}{"Title": "1"}))
	require.Equal(t, "OK-1", trim(buf.String()))
}
