package django

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func trim(str string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
}

func Test_Django_Render(t *testing.T) {
	engine := New("./views", ".django")
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Partials
	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect := `<h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2>`

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

func Test_Django_Layout(t *testing.T) {
	engine := New("./views", ".django")
	engine.Debug(true)
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	expect := `<!DOCTYPE html> <html> <head> <title>Main</title> </head> <body> <h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2> </body> </html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}

	// Nested Layout
	buf.Reset()
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main", "layouts/nested")
	expect = `<!DOCTYPE html> <html> <head> <title>Main</title> </head> <body> <h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2> </body> </html><div id="nest"><h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2></div>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}
