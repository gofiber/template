package html

import (
	"bytes"
	"io/ioutil"
	"net/http"
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

func Test_AddFunc(t *testing.T) {
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

func Test_Layout(t *testing.T) {
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

//Test_Layout_Multi checks if the layout can be rendered multiple times
func Test_Layout_Multi(t *testing.T) {
	engine := New("./views", ".html")

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	for i := 0; i < 2; i++ {
		var buf bytes.Buffer
		err := engine.Render(&buf, "index", map[string]interface{}{
			"Title": "Hello, World!",
		}, "layouts/main")
		expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
		result := trim(buf.String())
		if expect != result {
			t.Fatalf("\nExpected:\n%s\nResult:\n%s\n\nError: %s", expect, result, err)
		}
	}

}


func Test_FileSystem(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".html")

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

func Test_Reload(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".html")
	engine.Reload(true) // Optional. Default: false

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == "admin"
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	if err := ioutil.WriteFile("./views/reload.html", []byte("after reload\n"), 0644); err != nil {
		t.Fatalf("write file: %v\n", err)
	}
	defer func() {
		if err := ioutil.WriteFile("./views/reload.html", []byte("before reload\n"), 0644); err != nil {
			t.Fatalf("write file: %v\n", err)
		}
	}()

	engine.Load()

	var buf bytes.Buffer
	engine.Render(&buf, "reload", nil)
	expect := "after reload"
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}
