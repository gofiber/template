package django

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const xssPayload = `<script>alert(1)</script>`

func Test_XSS_DefaultAutoEscape(t *testing.T) {
	engine := New("./views", ".django")
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
	engine := New("./views", ".django")
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

func Test_HelperOutputIsEscaped(t *testing.T) {
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}()

	err = os.WriteFile(dir+"/helper.django", []byte(`<p>{{ helper() }}</p>`), 0o600)
	require.NoError(t, err)

	engine := New(dir, ".django")
	engine.AddFunc("helper", func() string {
		return xssPayload
	})
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err = engine.Render(&buf, "helper", nil)
	require.NoError(t, err)

	result := trim(buf.String())
	require.NotContains(t, result, xssPayload)
	require.Contains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}
