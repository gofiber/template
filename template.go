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
	ShouldReload() bool
	Loaded() bool
	SetLoaded(enabled bool) IEngineCore
	LockAndSetLoaded(enabled bool) IEngineCore
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
	// lock for funcmap and templates
	Mutex sync.RWMutex
	// template funcmap
	Funcmap map[string]interface{}
	// debug prints the parsed templates
	verbose bool
	// determines if the engine parsed all templates
	loaded bool
	// reload on each render
	shouldReload bool
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn interface{}) IEngineCore {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.Funcmap[name] = fn
	return e
}

// AddFuncMap adds the functions from a map to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFuncMap(m map[string]interface{}) IEngineCore {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	for name, fn := range m {
		e.Funcmap[name] = fn
	}
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) IEngineCore {
	e.verbose = enabled
	return e
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: "{{" and "}}".
func (e *Engine) Delims(left, right string) IEngineCore {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.Left, e.Right = left, right
	return e
}

// FuncMap returns the template's function map.
func (e *Engine) FuncMap() map[string]interface{} {
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()
	return e.Funcmap
}

// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) IEngineCore {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.LayoutName = key
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) IEngineCore {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.shouldReload = enabled
	return e
}

// ShouldReload returns true if the templates should be reloaded
func (e *Engine) ShouldReload() bool {
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()
	return e.shouldReload
}

// Loaded returns true if templates are loaded
func (e *Engine) Loaded() bool {
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()
	return e.loaded
}

// SetLoaded sets the loaded status
func (e *Engine) SetLoaded(enabled bool) IEngineCore {
	e.loaded = enabled
	return e
}

// LockAndSetLoaded locks and sets the loaded status
func (e *Engine) LockAndSetLoaded(enabled bool) IEngineCore {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	return e.SetLoaded(enabled)
}

// Verbose returns the verbose status
func (e *Engine) Verbose() bool {
	return e.verbose
}
