package slim

import (
	"bytes"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.Replace(trimmed, " <", "<", -1)
	trimmed = strings.Replace(trimmed, "> ", ">", -1)
	return trimmed
}

func Test_Render(t *testing.T) {
	engine := New("./views", ".slim")
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}
	// Partials
	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect := `<div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div>`

	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Single
	buf.Reset()
	engine.Render(&buf, "errors/404", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect = `<h1>Hello, World!</h1>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Layout(t *testing.T) {
	engine := New("./views", ".slim")

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	expect := `<!doctype html><html><head><title>Main</title></head><body><div><div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div></div></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Empty_Layout(t *testing.T) {
	engine := New("./views", ".slim")

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "")
	expect := `<div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_FileSystem(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".slim")

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	expect := `<!doctype html><html><head><title>Main</title></head><body><div><div><h2>Header</h2></div><h1>Hello, World!</h1><div><h2>Footer</h2></div></div></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Reload(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".slim")
	engine.Reload(true) // Optional. Default: false

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	if err := os.WriteFile("./views/reload.slim", []byte("p after reload\n"), 0644); err != nil {
		t.Fatalf("write file: %v\n", err)
	}
	defer func() {
		if err := os.WriteFile("./views/reload.slim", []byte("p before reload\n"), 0644); err != nil {
			t.Fatalf("write file: %v\n", err)
		}
	}()

	engine.Load()

	var buf bytes.Buffer
	engine.Render(&buf, "reload", nil)
	expect := "<p>after reload</p>"
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Benchmark_Slim(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	//expectExtended := `<!DOCTYPE html><html><head><title>Title</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
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
