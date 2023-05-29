package mustache

import (
	"bytes"
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
	engine := New("./views", ".mustache")
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}
	// Partials
	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}); err != nil {
		t.Fatal("Test_Render: failed to render index")
	}
	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Single
	buf.Reset()
	if err := engine.Render(&buf, "errors/404", fiber.Map{
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
	engine := New("./views", ".mustache")
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}, "layouts/main"); err != nil {
		t.Fatalf("render: %v", err)
	}
	expect := `<!DOCTYPE html><html><head><title>Hello, World!</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystemPartials(http.Dir("./views"), ".mustache", http.Dir("."))
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", fiber.Map{
		"Title": "Hello, World!",
	}, "layouts/main")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	expect := `<!DOCTYPE html><html><head><title>Hello, World!</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".mustache")
	engine.Reload(true) // Optional. Default: false

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	if err := os.WriteFile("./views/reload.mustache", []byte("after reload\n"), 0o600); err != nil {
		t.Fatalf("write file: %v\n", err)
	}
	defer func() {
		if err := os.WriteFile("./views/reload.mustache", []byte("before reload\n"), 0o600); err != nil {
			t.Fatalf("write file: %v\n", err)
		}
	}()

	if err := engine.Load(); err != nil {
		t.Fatal("engine failed to load")
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "reload", nil); err != nil {
		t.Fatal("Test_Reload: failed to render reload")
	}
	expect := "after reload"
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Benchmark_Mustache(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	engine := New("./views", ".mustache")

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
