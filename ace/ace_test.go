package ace

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
	engine := New("./views", ".ace")
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
	engine := New("./views", ".ace")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".ace")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	engine.Debug(true)
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
	engine := New("./views", ".ace")
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

	// The layout template on its own must not reach the closure the render
	// above installed - outside a layout render the layout function reports.
	var second bytes.Buffer
	require.Error(t, engine.Render(&second, "layouts/main", map[string]interface{}{"Title": "SECOND"}))
	require.Equal(t, before, first.String(), "a finished render's writer was written to again")
	require.NotContains(t, second.String(), "FIRST", "an earlier render's body leaked into a later one")

	// Under -race, neither render may write through the other's closure - and a
	// clean race report alone would still miss a page render left short of a body.
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
	engine := NewFileSystem(http.Dir("./views"), ".ace")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

// goland:noinspection GoDeprecation
func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".ace")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	engine.Reload(true) // Optional. Default: false

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/ShouldReload.ace", []byte("after ShouldReload\n"), 0o600)
	require.NoError(t, err)

	defer func() {
		err := os.WriteFile("./views/ShouldReload.ace", []byte("before ShouldReload\n"), 0o600)
		require.NoError(t, err)
	}()

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "ShouldReload", nil)
	require.NoError(t, err)

	expect := "<after>ShouldReload</after>"
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
	err = os.WriteFile(dir+"/func_map.ace", []byte(`
h2 {{lower .Var1}}
p {{upper .Var2}}`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".ace")

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

func Benchmark_Ace(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".ace")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
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

	b.Run("extended", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			var buf bytes.Buffer
			//nolint:gosec,errcheck // Return value not needed for benchmark
			_ = engine.Render(&buf, "extended", map[string]interface{}{
				"User": admin,
			}, "layouts/main")
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

	b.Run("extended_asserted", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			var buf bytes.Buffer
			err := engine.Render(&buf, "extended", map[string]interface{}{
				"User": admin,
			}, "layouts/main")

			require.NoError(b, err)
			require.Equal(b, expectExtended, trim(buf.String()))
		}
	})
}

func Benchmark_Ace_Parallel(b *testing.B) {
	expectSimple := `<h1>Hello, Parallel!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".ace")
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

	err = os.WriteFile(dir+"/child.ace", []byte("| C"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.ace", []byte("| L[{{embed}}]L"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/page.ace", []byte("| X{{partial}}Y"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".ace")
	// A template function that renders through the engine again must not
	// deadlock on a lock its own render holds.
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
		require.Equal(t, "XL[C]LY", trim(buf.String()))
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

	err = os.WriteFile(dir+"/self.ace", []byte("| S{{embed}}S"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.ace", []byte("| L[{{embed}}]L"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".ace")
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

	err = os.WriteFile(dir+"/plain.ace", []byte("| A-{{embed}}-B"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/child.ace", []byte("| C"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.ace", []byte("| L[{{embed}}]L"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".ace")
	// Overwriting a default action is documented as legal, so a layout render
	// must not clobber the override for later plain renders.
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
	engine := New(views, ".ace")
	require.Error(t, engine.Load())

	// A failed load must not stick: once the cause is gone, the next render
	// has to reload and serve - Load stays failed-state aware on its own.
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(views+"/index.ace", []byte("| OK"), 0o600))

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "index", nil))
	require.Equal(t, "OK", trim(buf.String()))
}
