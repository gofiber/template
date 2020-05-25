package pug

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func trim(str string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
}

func Test_Pug_Render(t *testing.T) {
	engine := New("./views", ".pug")
	// Partials
	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect := `<h2>Header</h2> <h1>Hello, World!</h1> <h2>Footer</h2>`

	result := strings.ReplaceAll(trim(buf.String()), " </h1>", "</h1>")
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
