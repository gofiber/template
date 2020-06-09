package html

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func trim(str string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
}

func Test_HTML_Render(t *testing.T) {
	engine := New("./views", ".html")
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
	// Layout
	buf.Reset()
	engine.Render(&buf, "index", nil, "layouts/main")
	expect = `<!DOCTYPE html> <html> <head> <title>Main</title> </head> <body> <h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2> </body> </html>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Nested Layout
	buf.Reset()
	engine.Render(&buf, "index", nil, "layouts/main", "layouts/nest")
	expect = `<!DOCTYPE html> <html> <head> <title>Main</title> </head> <body> <div id="nest"> <h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2> </div> </body> </html>`
	result = trim(buf.String())
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
