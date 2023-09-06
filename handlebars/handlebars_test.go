package handlebars

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".hbs")
	require.NoError(t, engine.Load())

	// Partials
	var buf bytes.Buffer
	err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)

	// Single
	buf.Reset()
	err = engine.Render(&buf, "errors/404", fiber.Map{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect = `<h1>Hello, World!</h1>`
	result = trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".hbs")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Hello, World!</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".hbs")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}, "")
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".hbs")
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Hello, World!</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".hbs")
	engine.Reload(true) // Optional. Default: false

	// Test Load() does not re-bind custom helpers
	engine.AddFunc("testHelper", func() string {
		return "Hello World"
	})
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/ShouldReload.hbs", []byte("after ShouldReload\n"), 0o600)
	require.NoError(t, err)

	defer func() {
		err := os.WriteFile("./views/ShouldReload.hbs", []byte("before ShouldReload\n"), 0o600)
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
	// Create a temporary Directory
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	// Create a temporary template file.
	err = os.WriteFile(dir+"/func_map.hbs", []byte(`<h2>{{lower Var1}}</h2><p>{{upper Var2}}</p>`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".hbs")
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

func Benchmark_Handlebars(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".hbs")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})

	var buf bytes.Buffer
	var err error

	b.Run("simple", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err = engine.Render(&buf, "simple", map[string]interface{}{
				"Title": "Hello, World!",
			})
		}

		require.NoError(b, err)
		require.Equal(b, expectSimple, trim(buf.String()))
	})

	b.Run("extended", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err = engine.Render(&buf, "extended", map[string]interface{}{
				"User":  "admin",
				"Title": "Main",
			}, "layouts/main")
		}

		require.NoError(b, err)
		require.Equal(b, expectExtended, trim(buf.String()))
	})
}
