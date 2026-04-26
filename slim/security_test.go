package slim

import (
	"bytes"
	"os"
	"testing"

	goslim "github.com/mattn/go-slim"
	"github.com/stretchr/testify/require"
)

const xssPayload = `<script>alert(1)</script>`

func Test_XSS(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".slim")
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
	engine := New("./views", ".slim")
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

func Test_HelperOutputIsEscaped(t *testing.T) {
	t.Parallel()
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/helper.slim", []byte(`p = helper()`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".slim")
	engine.AddFunc("helper", func(...goslim.Value) (goslim.Value, error) {
		return xssPayload, nil
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "helper", nil)
	require.NoError(t, err)

	result := trim(buf.String())
	require.NotContains(t, result, xssPayload)
	require.Contains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}
