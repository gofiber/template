package pug

import (
	"bytes"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

const (
	admin = "admin"
)

func trim(str string) string {
	trimmed := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(str, " "))
	trimmed = strings.ReplaceAll(trimmed, " <", "<")
	trimmed = strings.ReplaceAll(trimmed, "> ", ">")
	return trimmed
}

func Test_Render(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}
	// Partials
	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]any{
		"Title": "Hello, World!",
	}); err != nil {
		t.Fatal("Test_Render: failed to render index")
	}
	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`

	result := strings.ReplaceAll(trim(buf.String()), " </h1>", "</h1>")
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
	// Single
	buf.Reset()
	if err := engine.Render(&buf, "errors/404", map[string]any{
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
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]any{
		"Title": "Hello, World!",
	}, "layouts/main"); err != nil {
		t.Fatal("Test_Layout: failed to render index")
	}
	expect := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Empty_Layout(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]any{
		"Title": "Hello, World!",
	}, ""); err != nil {
		t.Fatal("Test_Empty_Layout: failed to render index")
	}
	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_FileSystem(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "index", map[string]any{
		"Title": "Hello, World!",
	}, "layouts/main"); err != nil {
		t.Fatal("Test_FileSystem: failed to render index")
	}
	expect := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}

func Test_Reload(t *testing.T) {
	t.Parallel()
	engine := NewFileSystem(http.Dir("./views"), ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	engine.Reload(true) // Optional. Default: false
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	if err := os.WriteFile("./views/reload.pug", []byte("after reload\n"), 0o600); err != nil {
		t.Fatalf("write file: %v\n", err)
	}
	defer func() {
		if err := os.WriteFile("./views/reload.pug", []byte("before reload\n"), 0o600); err != nil {
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
	expect := "<after>reload</after>"
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
		t.Fatal("failed to create a temporary directory")
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal("failed to remove the temporary directory")
		}
	}()

	// Create a temporary template file.
	if err = os.WriteFile(dir+"/func_map.pug", []byte(`
	h2 #{lower .Var1}
	p #{upper .Var2}`), 0o600); err != nil {
		t.Fatal("failed to write to func_map.pug")
	}

	engine := New(dir, ".pug")

	fm := map[string]any{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
	}

	engine.AddFuncMap(fm)

	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	if err := engine.Render(&buf, "func_map", map[string]any{
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

func Benchmark_Pug(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title><meta name="viewport" content="width=device-width, initial-scale=1, minimum-scale=1"/></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".pug")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	var buf bytes.Buffer
	var err error

	b.Run("simple", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err = engine.Render(&buf, "simple", map[string]any{
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

	b.Run("extended", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err = engine.Render(&buf, "extended", map[string]any{
				"User": admin,
			}, "layouts/main")
		}

		if err != nil {
			bb.Fatalf("Failed to render: %v", err)
		}
		result := trim(buf.String())
		if expectExtended != result {
			bb.Fatalf("Expected:\n%s\nResult:\n%s\n", expectExtended, result)
		}
	})
}
