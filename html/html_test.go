package html

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
	admin         = "admin"
	complexexpect = `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")
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
		"Error": "404 Not Found!",
	})
	require.NoError(t, err)

	expect = `<h1>404 Not Found!</h1>`
	result = trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_AddFunc(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	// Func is admin
	var buf bytes.Buffer
	err := engine.Render(&buf, admin, map[string]interface{}{
		"User": admin,
	})
	require.NoError(t, err)

	expect := `<h1>Hello, Admin!</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)

	// Func is not admin
	buf.Reset()
	err = engine.Render(&buf, admin, map[string]interface{}{
		"User": "john",
	})
	require.NoError(t, err)

	expect = `<h1>Access denied!</h1>`
	result = trim(buf.String())
	require.Equal(t, expect, result)

	// FuncMap
	fm := engine.FuncMap()
	_, ok := fm["isAdmin"]
	require.True(t, ok)
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
	err = os.WriteFile(dir+"/func_map.html", []byte(`<h2>{{lower .Var1}}</h2><p>{{upper .Var2}}</p>`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".html")
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

func Test_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	result := trim(buf.String())
	require.Equal(t, complexexpect, result)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")

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

// Test_Layout_Multi checks if the LayoutName can be rendered multiple times
func Test_Layout_Multi(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	for i := 0; i < 2; i++ {
		var buf bytes.Buffer
		err := engine.Render(&buf, "index", map[string]interface{}{
			"Title": "Hello, World!",
		}, "layouts/main")
		require.NoError(t, err)

		result := trim(buf.String())
		require.Equal(t, complexexpect, result)
	}
}

func Test_Layout_Nested(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/nested/main", "layouts/nested/base")
	require.NoError(t, err)

	result := trim(buf.String())
	require.Equal(t, complexexpect, result)
}

func Test_Layout_Isolation(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")
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

func Test_Layout_Isolation_Unreached(t *testing.T) {
	t.Parallel()

	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/page.html", []byte("BODY-{{.Secret}}"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/cond.html", []byte("<cond>{{if .Embed}}{{embed}}{{end}}</cond>"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".html")
	require.NoError(t, engine.Load())

	// A layout that renders without reaching the layout action still has to
	// leave the func map - which the whole set shares - clean behind it.
	var first bytes.Buffer
	require.NoError(t, engine.Render(&first, "page", map[string]interface{}{
		"Secret": "PRIVATE",
		"Embed":  false,
	}, "cond"))
	before := first.String()
	require.NotContains(t, before, "PRIVATE")

	var second bytes.Buffer
	//nolint:errcheck // only the isolation of the two renders is under test
	_ = engine.Render(&second, "cond", map[string]interface{}{"Embed": true})
	require.Equal(t, before, first.String(), "a finished render's writer was written to again")
	require.NotContains(t, second.String(), "PRIVATE", "an earlier render's binding leaked into a later one")
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".html")

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	result := trim(buf.String())
	require.Equal(t, complexexpect, result)
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".html")
	engine.Reload(true) // Optional. Default: false

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/ShouldReload.html", []byte("after ShouldReload\n"), 0o600)
	require.NoError(t, err)
	defer func() {
		err := os.WriteFile("./views/ShouldReload.html", []byte("before ShouldReload\n"), 0o600)
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

func Benchmark_Html(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".html")
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

			require.NoError(b, err)
			require.Equal(b, expectSimple, trim(buf.String()))
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

			require.NoError(b, err)
			require.Equal(b, expectExtended, trim(buf.String()))
		}
	})
}

func Benchmark_Html_Parallel(b *testing.B) {
	expectSimple := `<h1>Hello, Parallel!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".html")
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

	err = os.WriteFile(dir+"/child.html", []byte("C"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.html", []byte("L[{{embed}}]L"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/page.html", []byte("X{{partial}}Y"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".html")
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

	err = os.WriteFile(dir+"/self.html", []byte("S{{embed}}S"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.html", []byte("L[{{embed}}]L"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".html")
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

	err = os.WriteFile(dir+"/plain.html", []byte("A-{{embed}}-B"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/child.html", []byte("C"), 0o600)
	require.NoError(t, err)
	err = os.WriteFile(dir+"/layout.html", []byte("L[{{embed}}]L"), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".html")
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
	engine := New(views, ".html")
	require.Error(t, engine.Load())

	// A failed load must not stick: once the cause is gone, the next load
	// parses and renders.
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(views+"/index.html", []byte("OK"), 0o600))
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "index", nil))
	require.Equal(t, "OK", trim(buf.String()))
}
