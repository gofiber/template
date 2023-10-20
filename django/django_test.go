//nolint:paralleltest // running these in parallel causes a data race
package django

import (
	"bytes"
	"embed"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/flosch/pongo2/v6"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	engine := New("./views", ".django")
	require.NoError(t, engine.Load())

	// Partials
	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)

	// Single
	buf.Reset()
	err = engine.Render(&buf, "errors/404", map[string]interface{}{
		"Title": "Hello, World!",
	})
	require.NoError(t, err)

	expect = `<h1>Hello, World!</h1>`
	result = trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Layout(t *testing.T) {
	engine := New("./views", ".django")
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Empty_Layout(t *testing.T) {
	engine := New("./views", ".django")
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "")
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Invalid_Identifiers(t *testing.T) {
	engine := New("./views", ".django")
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title":               "Hello, World!",
		"Invalid.Identifiers": "Don't return error from checkForValidIdentifiers!",
	}, "")
	require.NoError(t, err)

	expect := `<h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_GetPongoBinding(t *testing.T) {
	// Test with pongo2.Context
	ctx := pongo2.Context{"key1": "value1"}
	result := getPongoBinding(ctx)
	assert.Equal(t, ctx, result, "Expected the same context")

	// Test with map[string]interface{}
	mapBinding := map[string]interface{}{"key2": "value2"}
	result = getPongoBinding(mapBinding)
	assert.Equal(t, pongo2.Context(mapBinding), result, "Expected the same context")

	// Test with fiber.Map
	fiberMap := fiber.Map{"key3": "value3"}
	result = getPongoBinding(fiberMap)
	assert.Equal(t, pongo2.Context(fiberMap), result, "Expected the same context")

	// Test with unsupported type
	result = getPongoBinding("unsupported")
	assert.Nil(t, result, "Expected nil for unsupported type")

	// Test with invalid key
	invalidCtx := pongo2.Context{"key1": "value1", "invalid.key": "value2"}
	result = getPongoBinding(invalidCtx)
	assert.Equal(t, pongo2.Context{"key1": "value1"}, result, "Expected the same context")
}

func Test_IsValidKey(t *testing.T) {
	assert.True(t, isValidKey("key1"), "Expected true for valid key")
	assert.False(t, isValidKey("invalid.key"), "Expected false for invalid key")
	assert.False(t, isValidKey("invalid-key"), "Expected false for invalid key")
	assert.False(t, isValidKey("key1\n"), "Expected false for invalid key")
	assert.False(t, isValidKey("key1 "), "Expected false for invalid key")
	assert.False(t, isValidKey("👍"), "Expected false for invalid key")
	assert.False(t, isValidKey("你好"), "Expected false for invalid key")

	// do sume fuzzing where we generate 1000 random strings and check if they are valid keys
	// valid keys match the following regex: [a-zA-Z0-9_]+
	reValidkeys := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	for i := 0; i < 1000; i++ {
		key := generateRandomString(10)
		assert.Equal(t, reValidkeys.MatchString(key), isValidKey(key), "Expected the same result for key")
	}
}

// generateRandomString generates a random string of length n
// with printable, non-whitespace characters
//
// helper function for Test_IsValidKey
func generateRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		for {
			c := rune(rand.Intn(0x10FFFF))                 // generate a random rune
			if !unicode.IsSpace(c) && unicode.IsPrint(c) { // check if it's a printable, non-whitespace character
				b[i] = c
				break
			}
		}
	}
	return string(b)
}

func Test_FileSystem(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".django")
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
	}, "layouts/main")
	require.NoError(t, err)

	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_AddFunc(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".django")
	engine.Debug(true)

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, admin, map[string]interface{}{
		"user": admin,
	},
	)
	require.NoError(t, err)

	expect := `<h1>Hello, Admin!</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_Reload(t *testing.T) {
	engine := NewFileSystem(http.Dir("./views"), ".django")
	engine.Reload(true) // Optional. Default: false

	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	err := os.WriteFile("./views/ShouldReload.django", []byte("after ShouldReload\n"), 0o600)
	require.NoError(t, err)

	defer func() {
		err := os.WriteFile("./views/ShouldReload.django", []byte("before ShouldReload\n"), 0o600)
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

//go:embed views
var viewsAsssets embed.FS

func Test_PathForwardingFileSystem(t *testing.T) {
	engine := NewPathForwardingFileSystem(http.FS(viewsAsssets), "/views", ".django")
	engine.Debug(true)
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "descendant", map[string]interface{}{
		"greeting": "World",
	})
	require.NoError(t, err)

	expect := `<h1>Hello World! from ancestor</h1>`
	result := trim(buf.String())
	require.Equal(t, expect, result)
}

func Test_AddFuncMap(t *testing.T) {
	// Create a temporary directory
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	// Create a temporary template file.
	err = os.WriteFile(dir+"/func_map.django", []byte(`<h2>{{Var1|lower}}</h2><p>{{Var2|upper}}</p>`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".django")
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

func Test_Invalid_Template(t *testing.T) {
	engine := New("./views", ".django")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "invalid", nil)
	require.Error(t, err)
}

func Test_Invalid_Layout(t *testing.T) {
	engine := New("./views", ".django")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", nil, "invalid")
	require.Error(t, err)
}

func Benchmark_Django(b *testing.B) {
	expectSimple := `<h1>Hello, World!</h1>`
	expectExtended := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, Admin!</h1><h2>Footer</h2></body></html>`
	engine := New("./views", ".django")
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
				"User": admin,
			}, "layouts/main")
		}

		require.NoError(b, err)
		require.Equal(b, expectExtended, trim(buf.String()))
	})

	b.Run("simple_with_invalid_binding_keys", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err = engine.Render(&buf, "simple", map[string]interface{}{
				"Title":       "Hello, World!",
				"Invalid_Key": "Don't return error from checkForValidIdentifiers!",
			})
		}

		require.NoError(b, err)
		require.Equal(b, expectSimple, trim(buf.String()))
	})

	b.Run("extended_with_invalid_binding_keys", func(bb *testing.B) {
		bb.ReportAllocs()
		bb.ResetTimer()
		for i := 0; i < bb.N; i++ {
			buf.Reset()
			err = engine.Render(&buf, "extended", map[string]interface{}{
				"User":        admin,
				"Invalid_Key": "Don't return error from checkForValidIdentifiers!",
			}, "layouts/main")
		}

		require.NoError(b, err)
		require.Equal(b, expectExtended, trim(buf.String()))
	})
}
