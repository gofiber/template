package html

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const xssPayload = `<script>alert(1)</script>`

func Test_XSS(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "simple", map[string]interface{}{
		"Title": xssPayload,
	})
	require.NoError(t, err)

	result := trim(buf.String())
	require.NotContains(t, result, xssPayload)
	require.Contains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func Test_Layout_DoesNotTrustUserProvidedEmbed(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".html")
	engine.AddFunc("isAdmin", func(user string) bool {
		return user == admin
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", map[string]interface{}{
		"Title": "Hello, World!",
		"embed": xssPayload,
	}, "layouts/main")
	require.NoError(t, err)

	result := trim(buf.String())
	require.Contains(t, result, "Hello, World!")
	require.NotContains(t, result, xssPayload)
	require.NotContains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func Test_ContextualEscaping(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/contexts.html", []byte(
		`<a href="{{.URL}}" title="{{.Attr}}">{{.Body}}</a><script>var msg = "{{.JS}}";</script>`,
	), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".html")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "contexts", map[string]interface{}{
		"URL":  "javascript:alert(1)",
		"Attr": `" onmouseover="alert(1)`,
		"Body": xssPayload,
		"JS":   `</script><script>alert(1)</script>`,
	})
	require.NoError(t, err)

	result := trim(buf.String())
	require.Contains(t, result, `href="#ZgotmplZ"`)
	require.NotContains(t, result, `javascript:alert(1)`)
	require.NotContains(t, result, xssPayload)
	require.NotContains(t, result, `onmouseover="alert(1)`)
	require.NotContains(t, result, `</script><script>alert(1)</script>`)
}
