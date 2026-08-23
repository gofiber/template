package template

import (
	"errors"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type customMap map[string]interface{}

func newEngine() *Engine {
	return &Engine{Funcmap: make(map[string]interface{})}
}

func Test_PreRenderCheck(t *testing.T) {
	t.Parallel()

	t.Run("not loaded", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		require.True(t, e.PreRenderCheck())
		require.False(t, e.Loaded)
	})

	t.Run("loaded", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		e.Loaded = true
		require.False(t, e.PreRenderCheck())
		require.True(t, e.Loaded)
	})

	t.Run("loaded with reload", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		e.Loaded = true
		e.Reload(true)
		require.True(t, e.PreRenderCheck())
		require.False(t, e.Loaded, "reloading has to clear the loaded flag")
	})

	t.Run("reload lifecycle", func(t *testing.T) {
		t.Parallel()
		e := newEngine()

		// Unloaded engine: keeps asking to be loaded.
		require.True(t, e.PreRenderCheck())
		e.Loaded = true

		// Loaded engine: answers false from here on, cached or not.
		require.False(t, e.PreRenderCheck())
		require.False(t, e.PreRenderCheck())

		// Turning reloading on has to take effect even though the previous
		// answer was cached, and has to stay in effect on every render.
		e.Reload(true)
		for range 3 {
			require.True(t, e.PreRenderCheck())
			require.False(t, e.Loaded)
			e.Loaded = true // as an engine's Load would
		}

		// And turning it back off has to settle again.
		e.Reload(false)
		require.False(t, e.PreRenderCheck())
		require.False(t, e.PreRenderCheck())
		require.True(t, e.Loaded)
	})

	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()
		e := newEngine()
		e.Loaded = true
		done := make(chan bool, 8)
		for range cap(done) {
			go func() { done <- e.PreRenderCheck() }()
		}
		for range cap(done) {
			require.False(t, <-done)
		}
	})
}

func Test_AcquireViewContext(t *testing.T) {
	t.Parallel()

	native := map[string]interface{}{"key": "value"}
	require.Equal(t, native, AcquireViewContext(native))

	custom := customMap{"key": "value"}
	require.Equal(t, map[string]interface{}{"key": "value"}, AcquireViewContext(custom))

	require.Equal(t, map[string]interface{}{"key": "value"}, AcquireViewContext(&custom))

	for name, binding := range map[string]interface{}{
		"nil":              nil,
		"nil pointer":      (*customMap)(nil),
		"nil map":          customMap(nil),
		"not a map":        "value",
		"non string keys":  map[int]string{1: "value"},
		"pointer to value": func() interface{} { v := "value"; return &v }(),
	} {
		require.Empty(t, AcquireViewContext(binding), name)
		require.NotNil(t, AcquireViewContext(binding), name)
	}
}

func Test_ViewContext_Range(t *testing.T) {
	t.Parallel()

	for name, binding := range map[string]interface{}{
		"native map": map[string]interface{}{"a": 1, "b": 2},
		"custom map": customMap{"a": 1, "b": 2},
		"pointer":    &customMap{"a": 1, "b": 2},
		"nil":        nil,
		"not a map":  "value",
		"empty":      customMap{},
	} {
		want := AcquireViewContext(binding)

		got := make(map[string]interface{}, ViewContextLen(binding))
		RangeViewContext(binding, func(key string, value interface{}) {
			got[key] = value
		})

		require.Equal(t, want, got, name)
		require.Len(t, want, ViewContextLen(binding), name)
	}
}

func Test_HasExtension(t *testing.T) {
	t.Parallel()

	require.True(t, HasExtension("views/index.html", ".html"))
	require.True(t, HasExtension("views/index.html.jet", ".html.jet"))
	require.True(t, HasExtension("a.html", ".html"))

	require.False(t, HasExtension(".html", ".html"), "a bare extension is not a template")
	require.False(t, HasExtension("views/index.htm", ".html"))
	require.False(t, HasExtension("views/index.html", ".jet"))
	require.False(t, HasExtension("", ".html"))
}

func Test_TemplateName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		directory string
		path      string
		extension string
		expect    string
	}{
		{"views", filepath.Join("views", "index.html"), ".html", "index"},
		{"./views", filepath.Join("views", "index.html"), ".html", "index"},
		{"views", filepath.Join("views", "partials", "footer.html"), ".html", "partials/footer"},
		{"/", "/index.html", ".html", "index"},
		{"views", filepath.Join("views", "index.html"), "", "index.html"},
		{"views", filepath.Join("views", "a.html.b.html"), ".html", "a.html.b"},
	} {
		name, err := TemplateName(tc.directory, tc.path, tc.extension)
		require.NoError(t, err)
		require.Equal(t, tc.expect, name)

		// The helper has to stay in step with what every engine open-coded
		// before: relative path, forward slashes, extension trimmed.
		rel, err := filepath.Rel(tc.directory, tc.path)
		require.NoError(t, err)
		require.Equal(t, strings.TrimSuffix(filepath.ToSlash(rel), tc.extension), name)
	}
}

func Test_TemplateName_Error(t *testing.T) {
	t.Parallel()

	_, err := TemplateName("relative", string(filepath.Separator)+"absolute/index.html", ".html")
	require.Error(t, err)
}

func Test_IsWord(t *testing.T) {
	t.Parallel()

	require.True(t, IsWord(""), "the empty string has no non-word byte")
	require.True(t, IsWord("key1"))
	require.True(t, IsWord("_Key_1"))
	require.False(t, IsWord("invalid.key"))
	require.False(t, IsWord("invalid-key"))
	require.False(t, IsWord("key1\n"))
	require.False(t, IsWord("key1 "))
	require.False(t, IsWord("👍"))
	require.False(t, IsWord("你好"))

	// Cross-check against the regexp for every length that matters to the
	// dispatch inside the SIMD kernels: below one machine word, across the
	// SWAR loop, and past the 32-byte and 128-byte vector thresholds.
	re := regexp.MustCompile(`^[a-zA-Z0-9_]*$`)
	alphabet := []rune("abzAZ09_.- \t\n/{}👍é")
	rnd := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test input
	for length := range 200 {
		for range 20 {
			var sb strings.Builder
			for range length {
				sb.WriteRune(alphabet[rnd.Intn(len(alphabet))])
			}
			key := sb.String()
			require.Equal(t, re.MatchString(key), IsWord(key), "key %q", key)
		}
	}
}

func Test_UnsafeConversions(t *testing.T) {
	t.Parallel()

	require.Equal(t, "hello", UnsafeString([]byte("hello")))
	require.Empty(t, UnsafeString(nil))

	require.Equal(t, []byte("hello"), UnsafeBytes("hello"))
	require.Empty(t, UnsafeBytes(""))
}

func Test_ReadFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "index.html")
	require.NoError(t, os.WriteFile(path, []byte("<h1>Hello</h1>"), 0o600))

	buf, err := ReadFile(path, nil)
	require.NoError(t, err)
	require.Equal(t, "<h1>Hello</h1>", string(buf))

	buf, err = ReadFile("index.html", http.Dir(dir))
	require.NoError(t, err)
	require.Equal(t, "<h1>Hello</h1>", string(buf))

	_, err = ReadFile(filepath.Join(dir, "missing.html"), nil)
	require.Error(t, err)
}

func Test_Walk(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "partials"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "partials", "footer.html"), []byte("footer"), 0o600))

	var names []string
	err := Walk(http.Dir(dir), "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() || !HasExtension(path, ".html") {
			return nil
		}
		name, err := TemplateName("/", path, ".html")
		if err != nil {
			return err
		}
		names = append(names, name)
		return nil
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"index", "partials/footer"}, names)

	err = Walk(http.Dir(dir), "/", func(_ string, _ os.FileInfo, _ error) error {
		return errors.New("boom")
	})
	require.Error(t, err)
}
