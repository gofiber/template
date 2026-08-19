package mustache

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const xssPayload = `<script>alert(1)</script>`

func Test_XSS(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "simple", customMap{
		"Title": xssPayload,
	})
	require.NoError(t, err)

	result := trim(buf.String())
	require.NotContains(t, result, xssPayload)
	require.Contains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func Test_Layout_DoesNotTrustUserProvidedEmbed(t *testing.T) {
	t.Parallel()
	engine := New("./views", ".mustache")
	require.NoError(t, engine.Load())

	var buf bytes.Buffer
	err := engine.Render(&buf, "index", customMap{
		"Title": "Hello, World!",
		"embed": xssPayload,
	}, "layouts/main")
	require.NoError(t, err)

	result := trim(buf.String())
	require.Contains(t, result, "Hello, World!")
	require.NotContains(t, result, xssPayload)
	require.NotContains(t, result, "&lt;script&gt;alert(1)&lt;/script&gt;")
}

func Test_PartialProvider_AllowsSafePaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "partials"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "partials", "header.mustache"), []byte("Header"), 0o600))

	provider := fileSystemPartialProvider{
		root:      dir,
		extension: ".mustache",
	}

	content, err := provider.Get("partials/header")
	require.NoError(t, err)
	require.Equal(t, "Header", content)
}

func Test_PartialProvider_RejectsPathTraversal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "safe.mustache"), []byte("safe"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.mustache"), []byte("secret"), 0o600))

	provider := fileSystemPartialProvider{
		fileSystem: http.Dir(dir),
		extension:  ".mustache",
	}

	content, err := provider.Get("safe")
	require.NoError(t, err)
	require.Equal(t, "safe", content)

	_, err = provider.Get("../secret")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid partial path")

	_, err = provider.Get("/secret")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid partial path")
}
