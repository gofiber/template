package django

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
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

func Test_AutoEscape_IsIsolatedPerEngine(t *testing.T) {
	unescaped := New("./views", ".django")
	unescaped.SetAutoEscape(false)
	require.NoError(t, unescaped.Load())

	escaped := New("./views", ".django")
	require.NoError(t, escaped.Load())

	var unescapedBuf bytes.Buffer
	err := unescaped.Render(&unescapedBuf, "simple", map[string]interface{}{
		"Title": xssPayload,
	})
	require.NoError(t, err)
	require.Contains(t, trim(unescapedBuf.String()), xssPayload)

	var escapedBuf bytes.Buffer
	err = escaped.Render(&escapedBuf, "simple", map[string]interface{}{
		"Title": xssPayload,
	})
	require.NoError(t, err)
	require.NotContains(t, trim(escapedBuf.String()), xssPayload)
	require.Contains(t, trim(escapedBuf.String()), "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func Test_Load_RejectsTraversalOutsideTemplateRoot(t *testing.T) {
	parentDir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.RemoveAll(parentDir))
	}()

	viewsDir := filepath.Join(parentDir, "views")
	require.NoError(t, os.MkdirAll(viewsDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, "secret.django"), []byte("secret"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(viewsDir, "index.django"), []byte(`{% include "../secret.django" %}`), 0o600))

	engine := New(viewsDir, ".django")
	err = engine.Load()
	require.Error(t, err)
}

func Test_PathForwardingFileSystem_RejectsTraversalOutsideTemplateRoot(t *testing.T) {
	parentDir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.RemoveAll(parentDir))
	}()

	viewsDir := filepath.Join(parentDir, "views")
	require.NoError(t, os.MkdirAll(viewsDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, "secret.django"), []byte("secret"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(viewsDir, "index.django"), []byte(`{% include "../secret.django" %}`), 0o600))

	engine := NewPathForwardingFileSystem(http.Dir(parentDir), "/views", ".django")
	err = engine.Load()
	require.Error(t, err)
}

func Test_Sandbox_BansRecommendedTagsAndFilters(t *testing.T) {
	dir, err := os.MkdirTemp(".", "")
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.django"), []byte(`{% include "partial.django" %}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "partial.django"), []byte(`<p>{{ Title|safe }}</p>`), 0o600))

	engine := New(dir, ".django")
	require.NoError(t, engine.Sandbox())

	err = engine.Load()
	require.Error(t, err)
}
