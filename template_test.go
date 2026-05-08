package template_test

import (
	"testing"

	template "github.com/gofiber/template/v2"
)

func TestEngine_AddFunc_InitializesFuncMap(t *testing.T) {
	t.Parallel()

	var engine template.Engine
	engine.AddFunc("hello", func() string {
		return "world"
	})

	if engine.Funcmap == nil {
		t.Fatal("expected Funcmap to be initialized")
	}

	if _, ok := engine.Funcmap["hello"]; !ok {
		t.Fatal("expected hello func to be registered")
	}
}

func TestEngine_AddFuncMap_InitializesFuncMap(t *testing.T) {
	t.Parallel()

	var engine template.Engine
	engine.AddFuncMap(map[string]interface{}{
		"hello": func() string {
			return "world"
		},
		"goodbye": func() string {
			return "moon"
		},
	})

	if len(engine.Funcmap) != 2 {
		t.Fatalf("expected 2 funcs, got %d", len(engine.Funcmap))
	}

	if _, ok := engine.Funcmap["hello"]; !ok {
		t.Fatal("expected hello func to be registered")
	}

	if _, ok := engine.Funcmap["goodbye"]; !ok {
		t.Fatal("expected goodbye func to be registered")
	}
}
