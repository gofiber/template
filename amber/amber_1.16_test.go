//go:build go1.16
// +build go1.16

//nolint:paralleltest // running these in parallel causes a data race
package amber

import (
	"bytes"
	"embed"
	"net/http"
	"testing"
)

//go:embed embed_views/*
var embededViews embed.FS

func Test_EmbedFileSystem(t *testing.T) {
	engine := NewFileSystem(http.FS(embededViews), ".amber")
	engine.Debug(true)
	if err := engine.Load(); err != nil {
		t.Fatalf("load: %v\n", err)
	}

	var buf bytes.Buffer
	err := engine.Render(&buf, "embed_views/embed", map[string]interface{}{
		"Title": "Hello, World!",
	}, "embed_views/layouts/main")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	expect := `<!DOCTYPE html><html><head><title>Main</title></head><body><h2>Header</h2><h1>Hello, World!</h1><h2>Footer</h2></body></html>`
	result := trim(buf.String())
	if expect != result {
		t.Fatalf("Expected:\n%s\nResult:\n%s\n", expect, result)
	}
}
