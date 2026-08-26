package pug

import (
	"bytes"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	admin = "admin"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	// Partials
	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := strings.ReplaceAll(trim(buf.String()), " </h1>", "</h1>")
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
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
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

func Test_Layout_Isolation(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var first bytes.Buffer
	require.NoError(t, engine.Render(&first, "index", map[string]interface{}{
		"Title": "FIRST",
	}, "layouts/main"))
	before := first.String()
	require.Contains(t, before, "FIRST")

	// A standalone layout render must not reach another render's closure.
	var second bytes.Buffer
	require.Error(t, engine.Render(&second, "layouts/main", map[string]interface{}{"Title": "SECOND"}))
	require.Equal(t, before, first.String(), "a finished render's writer was written to again")
	require.NotContains(t, second.String(), "FIRST", "an earlier render's body leaked into a later one")

	// Concurrent layout renders must not disturb page renders or their output.
	const rounds = 20
	errs := make([]error, rounds)
	outs := make([]string, rounds)
	layErrs := make([]error, rounds)
	var wg sync.WaitGroup
	for i := range rounds {
		wg.Add(2)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			errs[i] = engine.Render(&buf, "index", map[string]interface{}{"Title": "FIRST"}, "layouts/main")
			outs[i] = buf.String()
		}()
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			layErrs[i] = engine.Render(&buf, "layouts/main", map[string]interface{}{"Title": "SECOND"})
		}()
	}
	wg.Wait()

	for i := range rounds {
		require.NoError(t, errs[i])
		require.Equal(t, before, outs[i], "a concurrent layout render disturbed a page render")
		require.Error(t, layErrs[i], "a standalone layout render reached another render's closure")
	}
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	engine.Reload(true) // Optional. Default: false
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/reload.pug", []byte("after reload\n"), 0o600)
	require.NoError(t, err)

	defer func() {
		err := os.WriteFile("./views/reload.pug", []byte("before reload\n"), 0o600)
		require.NoError(t, err)
	}()

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "reload", nil)
	require.NoError(t, err)

	expect := "<after>reload</after>"
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
	err = os.WriteFile(dir+"/func_map.pug", []byte(`
	h2 #{lower .Var1}
	p #{upper .Var2}`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".pug")
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

func Benchmark_Pug(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
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
				"User": admin,
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
				"User": admin,
			}, "layouts/main")
			require.NoError(bb, err)
			require.Equal(bb, expectExtended, trim(buf.String()))
		}
	})
}

func Benchmark_Pug_Parallel(b *testing.B) {
	expectSimple := `<h1>Hello, Parallel!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
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
					"User": admin,
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
					"User": admin,
				}, "layouts/main")
				require.NoError(bb, err)
				require.Equal(bb, expectExtended, trim(buf.String()))
			}
		})
	})
}

func Test_Render_Reentrant(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/child.pug", []byte("| C"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.pug", []byte("| L[{{embed}}]L"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/page.pug", []byte("| X{{partial}}Y"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".pug")
	// A template function that calls Render again must not deadlock.
	engine.AddFunc("partial", func() (string, error) {
		var buf bytes.Buffer
		err := engine.Render(&buf, "child", nil, "layout")
		return buf.String(), err
	})
	require.NoError(t, engine.Load())

	done := make(chan error, 1)
	var buf bytes.Buffer
	go func() {
		done <- engine.Render(&buf, "page", nil)
	}()
	select {
	case err := <-done:
		require.NoError(t, err)
		// jade pads piped lines with newlines, so compare without whitespace.
		require.Equal(t, "XL[C]LY", strings.ReplaceAll(trim(buf.String()), " ", ""))
	case <-time.After(5 * time.Second):
		t.Fatal("re-entrant render deadlocked")
	}
}

func Test_Render_SelfEmbed_Errors(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/self.pug", []byte("| S{{embed}}S"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.pug", []byte("| L[{{embed}}]L"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".pug")
	require.NoError(t, engine.Load())

	// A page holding the layout action must fail its render, not recurse.
	var buf bytes.Buffer
	require.Error(t, engine.Render(&buf, "self", nil, "layout"))
}

func Test_AddFunc_Layout_Override(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/plain.pug", []byte("| A-{{embed}}-B"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/child.pug", []byte("| C"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.pug", []byte("| L[{{embed}}]L"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".pug")
	// A layout render must not clobber an AddFunc override of the layout function.
	engine.AddFunc(engine.LayoutName, func() string { return "CUSTOM" })
	require.NoError(t, engine.Load())

	var before bytes.Buffer
	require.NoError(t, engine.Render(&before, "plain", nil))
	require.Contains(t, before.String(), "CUSTOM")

	var lay bytes.Buffer
	require.NoError(t, engine.Render(&lay, "child", nil, "layout"))

	var after bytes.Buffer
	require.NoError(t, engine.Render(&after, "plain", nil))
	require.Equal(t, before.String(), after.String())
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
	engine := New(views, ".pug")
	require.Error(t, engine.Load())

	// Once the cause is gone, the next render has to reload and serve.
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(views+"/index.pug", []byte("| OK"), 0o600))

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "index", nil))
	require.Equal(t, "OK", trim(buf.String()))
}

func Test_Layout_Concurrent_Rename(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	// Layout writes LayoutName while layout renders read it.
	errs := make([]error, 4)
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				var buf bytes.Buffer
				if err := engine.Render(&buf, "index", map[string]interface{}{"Title": "Hello, World!"}, "layouts/main"); err != nil {
					errs[g] = err
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 200 {
			engine.Layout("embed")
		}
	}()
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}
