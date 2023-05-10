package template

import (
	"io"
)

// Engine interface, to be implemented for any templating engine added to the repository
type Engine interface {
	EngineInterface
	Load() error
	Render(out io.Writer, template string, binding interface{}, layout ...string) error
}

// EngineInterface interface
type EngineInterface interface {
	AddFunc(name string, fn interface{}) EngineInterface
	AddFuncMap(m map[string]interface{}) EngineInterface
	Debug(enabled bool) EngineInterface
	Delims(left, right string) EngineInterface
	FuncMap() map[string]interface{}
	Layout(key string) EngineInterface
	Reload(enabled bool) EngineInterface
}
