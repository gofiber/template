package mustache

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type customMap map[string]interface{}

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	// Partials
	var buf bytes.Buffer
	err := engine.Render(&buf, "index", customMap{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)

	// Single
	buf.Reset()
	err = engine.Render(&buf, "errors/404", customMap{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect = `<h1>Hello, World!</h1>`
	result = trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Render_PartialsFromNestedTemplate(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "nested/relative", customMap{
		"Title": "Hello, Nested!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, Nested!</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Render_RootAnchoredPartial(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "rooted", customMap{
		"Title": "Hello, Root!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, Root!</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Render_FullPathPartial(t *testing.T) {
	t.Parallel()

	views := filepath.Join(t.TempDir(), "views")
	require.NoError(t, os.MkdirAll(filepath.Join(views, "partials"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(views, "partials", "header.mustache"), []byte("<h2>Header</h2>"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(views, "index.mustache"), []byte("{{> views/partials/header }}<h1>{{Title}}</h1>"), 0o600))

	engine := New(views, ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", customMap{
		"Title": "Hello, Full!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, Full!</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_FileSystem_SeparatePartials(t *testing.T) {
	t.Parallel()
	engine := NewFileSystemPartials(http.Dir("./views/nested"), ".mustache", http.Dir("./views"))
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "relative", customMap{
		"Title": "Hello, Partials!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, Partials!</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Load_MissingPartial(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.mustache"), []byte("{{> partials/missing }}"), 0o600))

	engine := New(dir, ".mustache")
	err := engine.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, `views: template broken includes partial "partials/missing", which does not exist`)
}

func Test_Load_MissingPartialInsideSection(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.mustache"), []byte("{{#Show}}{{> nope }}{{/Show}}"), 0o600))

	engine := New(dir, ".mustache")
	err := engine.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, `includes partial "nope", which does not exist`)
}

func Test_Load_SelfIncludingPartial(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.mustache"), []byte("A{{> a }}"), 0o600))

	engine := New(dir, ".mustache")
	err := engine.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, "views: partial cycle: a -> a")
}

func Test_Load_PartialCycle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.mustache"), []byte("A{{> b }}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.mustache"), []byte("B{{> a }}"), 0o600))

	engine := New(dir, ".mustache")
	err := engine.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, "views: partial cycle: a -> b -> a")
}

func Test_Load_PartialOutsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	const marker = "outside-the-root"
	require.NoError(t, os.WriteFile(filepath.Join(root, "sibling.mustache"), []byte(marker), 0o600))

	views := filepath.Join(root, "views")
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(views, "escape.mustache"), []byte("{{> ../sibling }}"), 0o600))

	engine := New(views, ".mustache")
	err := engine.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, `includes partial "../sibling", which does not exist`)
	require.NotContains(t, err.Error(), marker)
}

func Test_Load_PartialThroughSymlink(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink needs a privilege the runner may not hold")
	}

	root := t.TempDir()
	const marker = "outside-the-root"
	outside := filepath.Join(root, "outside")
	require.NoError(t, os.MkdirAll(outside, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "creds.mustache"), []byte(marker), 0o600))

	views := filepath.Join(root, "views")
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.Symlink(outside, filepath.Join(views, "link")))
	require.NoError(t, os.WriteFile(filepath.Join(views, "page.mustache"), []byte("{{> link/creds }}"), 0o600))

	engine := New(views, ".mustache")
	err := engine.Load()
	require.Error(t, err)
	require.ErrorContains(t, err, `includes partial "link/creds", which does not exist`)
	require.NotContains(t, err.Error(), marker)
}

func Test_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", customMap{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Hello, World!</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", customMap{
		"Title": "Hello, World!",
	}, "")
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystemPartials(http.Dir("./views"), ".mustache", http.Dir("./views"))
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", customMap{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Hello, World!</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".mustache")
	engine.Reload(true) // Optional. Default: false
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/reload.mustache", []byte("after reload\n"), 0o600)
	require.NoError(t, err)

	defer func() {
		err := os.WriteFile("./views/reload.mustache", []byte("before reload\n"), 0o600)
		require.NoError(t, err)
	}()

	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "reload", nil)
	require.NoError(t, err)

	expect := "after reload"
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Benchmark_Mustache(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	engine := New("./views", ".mustache")
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
			require.NoError(bb, err)
			require.Equal(bb, expectSimple, trim(buf.String()))
		}
	})
}

func Benchmark_Mustache_Parallel(b *testing.B) {
	expectSimple := `<h1>Hello, Parallel!</h1>`
	engine := New("./views", ".mustache")
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

func Test_Render_Concurrent(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	var first bytes.Buffer
	require.NoError(t, engine.Render(&first, "index", map[string]interface{}{"Title": "C"}))
	before := first.String()

	// Concurrent renders pin mustache's executes-without-writes claim under -race.
	const rounds = 20
	errs := make([]error, rounds)
	outs := make([]string, rounds)
	var wg sync.WaitGroup
	for i := range rounds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			errs[i] = engine.Render(&buf, "index", map[string]interface{}{"Title": "C"})
			outs[i] = buf.String()
		}()
	}
	wg.Wait()

	for i := range rounds {
		require.NoError(t, errs[i])
		require.Equal(t, before, outs[i])
	}
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
	engine := New(views, ".mustache")
	require.Error(t, engine.Load())

	// Once the cause is gone, the next render has to reload and serve.
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(views+"/index.mustache", []byte(`OK-{{Title}}`), 0o600))

	var buf bytes.Buffer
	require.NoError(t, engine.Render(&buf, "index", map[string]interface{}{"Title": "1"}))
	require.Equal(t, "OK-1", buf.String())
}
