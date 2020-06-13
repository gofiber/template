package html

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
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

func Test_HTML_Render(t *testing.T) {
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Partials
	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Single
	buf.Reset()
	engine.Render(&buf, "errors/404", map[string]interface{}{
		"Error": "404 Not Found!",
	})
	expect = `<h1>404 Not Found!</h1>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_HTML_AddFunc(t *testing.T) {
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	// Func is admin
	var buf bytes.Buffer
	engine.Render(&buf, "admin", map[string]interface{}{
		"User": "admin",
	})
	expect := `<h1>Hello, Admin!</h1>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}

	// Func is not admin
	buf.Reset()
	engine.Render(&buf, "admin", map[string]interface{}{
		"User": "john",
	})
	expect = `<h1>Access denied!</h1>`
	result = trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_HTML_Layout(t *testing.T) {
	engine := New("./views", ".html")

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_HTML_Reload(t *testing.T) {
	engine := New("./views", ".html")
	engine.Reload(true)

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	if err != nil {
		t.Fatal(err)
	}

	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}

	index, err := ioutil.ReadFile(filepath.Join("views", "index.html"))
	if err != nil {
		t.Fatalf("failed to read views/index.html: %s\n", err)
	}

	reloadTmpl := filepath.Join("views", "reload_test.html")
	addition := []byte(`<h2>Hello World</h2>`)

	err = ioutil.WriteFile(reloadTmpl, append(index, addition...), 0600)
	if err != nil {
		t.Fatalf("failed to write %s: %s\n", reloadTmpl, err)
	}
	defer os.RemoveAll(reloadTmpl)

	buf.Reset()

	err = engine.Render(&buf, "reload_test", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(buf.Bytes(), addition) {
		t.Fatal("reloading template failed")
	}
}
