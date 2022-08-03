//go:build go1.16
// +build go1.16

package django

import (
	"bytes"
	"embed"
	"net/http"
	"testing"
)

//go:embed views
var viewsAsssets embed.FS

func Test_PathForwardingFileSystem(t *testing.T) {
	engine := NewPathForwardingFileSystem(http.FS(viewsAsssets), "/views", ".django")
	engine.Debug(true)
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	err := engine.Render(&buf, "descendant", map[string]interface{}{
		"greeting": "World",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	expect := `<h1>Hello World! from ancestor</h1>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}
