package mustache

import (
	"bytes"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

func Test_Render_RelativePartials(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	// nested/relative.mustache includes partials/header, so the partial has to
	// resolve against the engine directory rather than the including template.
	var buf bytes.Buffer
	err := engine.Render(&buf, "nested/relative", customMap{
		"Title": "Hello, Relative!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, Relative!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_LookupCandidates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseDir string
		partial string
		expect  []string
	}{
		{
			name:    "engine directory first",
			baseDir: "./views",
			partial: "partials/header",
			expect:  []string{"views/partials/header.mustache", "partials/header.mustache"},
		},
		{
			name:    "full path stays supported",
			baseDir: "./views",
			partial: "views/partials/header",
			expect:  []string{"views/views/partials/header.mustache", "views/partials/header.mustache"},
		},
		{
			name:    "extension is not doubled",
			baseDir: "./views",
			partial: "partials/header.mustache",
			expect:  []string{"views/partials/header.mustache", "partials/header.mustache"},
		},
		{
			name:    "leading dot slash is dropped",
			baseDir: "./views",
			partial: "./partials/header",
			expect:  []string{"views/partials/header.mustache", "partials/header.mustache"},
		},
		{
			name:    "filesystem root adds no prefix",
			baseDir: "/",
			partial: "partials/header",
			expect:  []string{"partials/header.mustache"},
		},
		{
			name:    "traversal is rejected",
			baseDir: "./views",
			partial: "../../../etc/passwd",
		},
		{
			name:    "traversal inside the path is rejected",
			baseDir: "./views",
			partial: "partials/../../../etc/passwd",
		},
		{
			name:    "absolute path is rejected",
			baseDir: "./views",
			partial: "/etc/passwd",
		},
		{
			name:    "blank partial is rejected",
			baseDir: "./views",
			partial: "   ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider := fileSystemPartialProvider{extension: ".mustache", baseDir: tc.baseDir}
			require.Equal(t, tc.expect, provider.lookupCandidates(tc.partial))
		})
	}
}

func Test_Render_MissingPartial(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.mustache"), []byte("{{> partials/missing }}"), 0o600))

	engine := New(dir, ".mustache")
	require.NoError(t, engine.Load())

	// A missing partial must not render empty: it has to name the partial and
	// every path that was tried.
	var buf bytes.Buffer
	err := engine.Render(&buf, "broken", nil)
	require.Error(t, err)
	require.ErrorContains(t, err, `partial "partials/missing" does not exist`)
	require.ErrorContains(t, err, "tried: "+path.Join(filepath.ToSlash(dir), "partials/missing.mustache")+", partials/missing.mustache")
}

func Test_Render_PartialPathTraversal(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	const secret = "top-secret"
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.mustache"), []byte(secret), 0o600))

	views := filepath.Join(root, "views")
	require.NoError(t, os.MkdirAll(views, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(views, "escape.mustache"), []byte("{{> ../secret }}"), 0o600))

	engine := New(views, ".mustache")
	require.NoError(t, engine.Load())

	// A partial must not read its way out of the engine directory.
	var buf bytes.Buffer
	err := engine.Render(&buf, "escape", nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "is not a valid path inside the template root")
	require.NotContains(t, err.Error(), secret)
	require.NotContains(t, buf.String(), secret)
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
