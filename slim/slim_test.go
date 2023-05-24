package slim

import (
	"bytes"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

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
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}
	// Partials
	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}); err != nil {
		t.Fatal("Test_Render: failed to render index")
	}
	expect := `<div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div>`

	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Single
	buf.Reset()
	if err := engine.Render(&buf, "errors/404", map[string]interface{}{
		"Title": "Hello, World!",
	}); err != nil {
		t.Fatal("Test_Render: failed to render 404")
	}
	expect = `<h1>Hello, World!</h1>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".slim")

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main"); err != nil {
		t.Fatal("Test_Layout: failed to render index")
	}
	expect := `<!doctype html><html><head><title>Main</title></head><body><div><div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div></div></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".slim")

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, ""); err != nil {
		t.Fatal("Test_Empty_Layout: failed to render index")
	}
	expect := `<div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".slim")

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main"); err != nil {
		t.Fatal("Test_FileSystem: failed to render index")
	}
	expect := `<!doctype html><html><head><title>Main</title></head><body><div><div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div></div></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".slim")
	engine.Reload(true) // Optional. Default: false

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	if err := os.WriteFile("./views/ShouldReload.slim", []byte("p after ShouldReload\n"), 0o600); err != nil {
		t.Fatalf("write file: %v\n", err)
	}
	defer func() {
		if err := os.WriteFile("./views/ShouldReload.slim", []byte("p before ShouldReload\n"), 0o600); err != nil {
			t.Fatalf("write file: %v\n", err)
		}
	}()

	if err := engine.Load(); err != nil {
		t.Fatal("engine failed to load")
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "ShouldReload", nil); err != nil {
		t.Fatal("Test_Reload: failed to render ShouldReload")
	}
	expect := "<p>after ShouldReload</p>"
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_AddFuncMap(t *testing.T) {
	t.Parallel()
	// Create a temporary directory
	dir, err := os.MkdirTemp(".", "")
	if err != nil {
		t.Fatal("failed to create temporary directory")
	}

	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal("failed to delete temporary directory")
		}
	}()

	// Create a temporary template file.
	if err = os.WriteFile(dir+"/func_map.slim", []byte(`
h2 = lower(Var1)
p = upper(Var2)`), 0o600); err != nil {
		t.Fatal("Test_AddFuncMap: failed to write to func_map.slim")
	}

	engine := New(dir, ".slim")

	fm := map[string]any{
		"lower": func(s ...slim.Value) (slim.Value, error) {
			slimvalue, ok := s[0].(string)
			if !ok {
				t.Fatal("failed type assertion on slim lower")
			}
			return strings.ToLower(slimvalue), nil
		},
		"upper": func(s ...slim.Value) (slim.Value, error) {
			slimvalue, ok := s[0].(string)
			if !ok {
				t.Fatal("failed type assertion on slim lower")
			}
			return strings.ToUpper(slimvalue), nil
		},
	}
	engine.AddFuncMap(fm)

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "func_map", map[string]interface{}{
		"Var1": "LOwEr",
		"Var2": "upPEr",
	}); err != nil {
		t.Fatal("Test_AddFuncMap: failed to render func_map")
	}
	expect := `<h2>lower</h2><p>UPPER</p>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}

	// FuncMap
	fm2 := engine.FuncMap()
	if _, ok := fm2["lower"]; !ok {
		t.Fatalf("Function lower does not exist in FuncMap().\n")
	}
	if _, ok := fm2["upper"]; !ok {
		t.Fatalf("Function upper does not exist in FuncMap().\n")
	}
}

func Benchmark_Slim(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	engine := New("./views", ".slim")
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

		if err != nil {
			bb.Fatalf("Failed to render: %v", err)
		}
		result := trim(buf.String())
		if expectSimple != result {
			bb.Fatalf("Expected:\n%s\nResult:\n%s\n", expectSimple, result)
		}
	})
}
