package template

import (
	"io"
	"net/http"
	"sync"
)

// IEngine interface, to be implemented for any templating engine added to the repository
type IEngine interface {
	IEngineCore
	Load() error
	Render(out io.Writer, template string, binding interface{}, layout ...string) error
}

// IEngineCore interface
type IEngineCore interface {
	AddFunc(name string, fn interface{}) IEngineCore
	AddFuncMap(m map[string]interface{}) IEngineCore
	Debug(enabled bool) IEngineCore
	Delims(left, right string) IEngineCore
	FuncMap() map[string]interface{}
	Layout(key string) IEngineCore
	Reload(enabled bool) IEngineCore
	PreRenderCheck() bool
}

// Engine engine struct
type Engine struct {
	IEngineCore
	// delimiters
	Left  string
	Right string
	// views folder
	Directory string
	// http.FileSystem supports embedded files
	FileSystem http.FileSystem
	// views extension
	Extension string
	// layout variable name that incapsulates the template
	LayoutName string
	// determines if the engine parsed all templates
	Loaded bool
	// reload on each render
	ShouldReload bool
	// debug prints the parsed templates
	Verbose bool
	// lock for funcmap and templates
	Mutex sync.RWMutex
	// template funcmap
	Funcmap map[string]interface{}
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn interface{}) IEngineCore {
	e.Mutex.Lock()
	e.Funcmap[name] = fn
	e.Mutex.Unlock()
	return e
}

// AddFuncMap adds the functions from a map to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFuncMap(m map[string]interface{}) IEngineCore {
	e.Mutex.Lock()
	for name, fn := range m {
		e.Funcmap[name] = fn
	}
	e.Mutex.Unlock()
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) IEngineCore {
	e.Mutex.Lock()
	e.Verbose = enabled
	e.Mutex.Unlock()
	return e
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: "{{" and "}}".
func (e *Engine) Delims(left, right string) IEngineCore {
	e.Mutex.Lock()
	e.Left, e.Right = left, right
	e.Mutex.Unlock()
	return e
}

// FuncMap returns the template's function map.
func (e *Engine) FuncMap() map[string]interface{} {
	return e.Funcmap
}

// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) IEngineCore {
	e.Mutex.Lock()
	e.LayoutName = key
	e.Mutex.Unlock()
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) IEngineCore {
	e.Mutex.Lock()
	e.ShouldReload = enabled
	e.Mutex.Unlock()
	return e
}

// Check if the engine should reload the templates before rendering
// Explicit Mute Unlock vs defer offers better performance
func (e *Engine) PreRenderCheck() bool {
	e.Mutex.Lock()

	if !e.Loaded || e.ShouldReload {
		if e.ShouldReload {
			e.Loaded = false
		}
		e.Mutex.Unlock()
		return true
	}
	e.Mutex.Unlock()
	return false
}
