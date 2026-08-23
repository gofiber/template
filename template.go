// Package template provides shared rendering primitives for Fiber template engines.
package template

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gofiber/utils/v2"
	"github.com/gofiber/utils/v2/simd"
)

// Reflect types the binding helpers compare against. Type identity in reflect
// is a pointer comparison, so these keep the check to a couple of loads.
var (
	stringType       = reflect.TypeOf("")
	anyType          = reflect.TypeOf((*interface{})(nil)).Elem()
	mapStringAnyType = reflect.TypeOf(map[string]interface{}(nil))
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
	// reload on each render, toggled through Reload
	ShouldReload bool
	// debug prints the parsed templates
	Verbose bool
	// lock for funcmap and templates
	Mutex sync.RWMutex
	// template funcmap
	Funcmap map[string]interface{}
	// ready mirrors "Loaded && !ShouldReload" so PreRenderCheck can answer the
	// steady state without touching Mutex at all. It is only ever set from
	// under Mutex, and cleared by Reload.
	ready atomic.Bool
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
//
// Toggle reloading through this method rather than by assigning ShouldReload:
// it also invalidates the state PreRenderCheck caches for its lock-free path.
func (e *Engine) Reload(enabled bool) IEngineCore {
	e.Mutex.Lock()
	e.ShouldReload = enabled
	e.Mutex.Unlock()
	e.ready.Store(false)
	return e
}

// PreRenderCheck determines if the engine should reload the templates before rendering.
// Explicit mutex unlock vs defer offers better performance.
//
// A loaded engine with reloading disabled - the steady state every render
// passes through - is answered from an atomic flag instead of the mutex.
// Taking even the shared lock here would make each render alternate between
// reader and writer on the mutex the layout paths need exclusively, which
// convoys badly once several goroutines render at once.
func (e *Engine) PreRenderCheck() bool {
	if e.ready.Load() {
		return false
	}

	e.Mutex.Lock()
	if !e.Loaded || e.ShouldReload {
		if e.ShouldReload {
			e.Loaded = false
		}
		e.Mutex.Unlock()
		return true
	}
	// Loaded with reloading off: the answer cannot change again until Reload
	// clears the flag, so let every later render skip the mutex.
	e.ready.Store(true)
	e.Mutex.Unlock()
	return false
}

// AcquireViewContext ensures the binding value is represented as a map[string]interface{}
// so template engines can safely inject layout specific data while preserving the
// original values. It supports native map[string]interface{} values as well as
// user-defined map types (for example fiber.Map) with string keys.
func AcquireViewContext(binding interface{}) map[string]interface{} {
	if binds, ok := binding.(map[string]interface{}); ok {
		return binds
	}

	val, ok := bindingMap(binding)
	if !ok {
		return make(map[string]interface{})
	}

	result := make(map[string]interface{}, val.Len())
	rangeMap(val, func(key string, value interface{}) {
		result[key] = value
	})
	return result
}

// ViewContextLen reports how many key/value pairs RangeViewContext yields for
// binding. Engines use it to size their own context type in a single
// allocation before filling it.
func ViewContextLen(binding interface{}) int {
	if binds, ok := binding.(map[string]interface{}); ok {
		return len(binds)
	}

	val, ok := bindingMap(binding)
	if !ok {
		return 0
	}
	return val.Len()
}

// RangeViewContext resolves binding exactly like AcquireViewContext and passes
// every key/value pair to fn. Engines that keep bindings in their own context
// type use it to fill that type directly, instead of building a
// map[string]interface{} only to copy out of it again.
func RangeViewContext(binding interface{}, fn func(key string, value interface{})) {
	if binds, ok := binding.(map[string]interface{}); ok {
		for key, value := range binds {
			fn(key, value)
		}
		return
	}

	val, ok := bindingMap(binding)
	if !ok {
		return
	}
	rangeMap(val, fn)
}

// directMap returns val as a plain map[string]interface{} when its type converts
// to one without copying - a named map[string]interface{} such as fiber.Map,
// which is what bindings overwhelmingly are. The second return value is false
// for any other map type, which has to go through reflect.
func directMap(val reflect.Value) (map[string]interface{}, bool) {
	typ := val.Type()
	if typ.Kind() != reflect.Map || typ.Key() != stringType || typ.Elem() != anyType {
		return nil, false
	}
	binds, ok := val.Convert(mapStringAnyType).Interface().(map[string]interface{})
	return binds, ok
}

// rangeMap hands every entry of the string-keyed map value val to fn. Where the
// map's type allows it, the entries are ranged over directly instead of through
// reflect's map iterator, which heap-allocates on every call.
func rangeMap(val reflect.Value, fn func(key string, value interface{})) {
	if binds, ok := directMap(val); ok {
		for key, value := range binds {
			fn(key, value)
		}
		return
	}

	iter := val.MapRange()
	for iter.Next() {
		fn(iter.Key().String(), iter.Value().Interface())
	}
}

// bindingMap resolves binding to the non-nil, string-keyed map value it wraps,
// dereferencing a pointer on the way. The second return value is false when
// binding holds no such map.
func bindingMap(binding interface{}) (reflect.Value, bool) {
	if binding == nil {
		return reflect.Value{}, false
	}

	val := reflect.ValueOf(binding)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return reflect.Value{}, false
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Map || val.IsNil() {
		return reflect.Value{}, false
	}

	if val.Type().Key().Kind() != reflect.String {
		return reflect.Value{}, false
	}
	return val, true
}

// HasExtension reports whether path is a template file for extension: it has
// to end in extension and carry a name in front of it. This is the check every
// engine applies to the files handed to it while walking the views directory.
func HasExtension(path, extension string) bool {
	return len(path) > len(extension) && strings.HasSuffix(path, extension)
}

// TemplateName derives the name a template file is registered under: its path
// relative to directory, with OS separators normalised to '/' and the
// extension trimmed. ./views/partials/footer.html becomes partials/footer.
func TemplateName(directory, path, extension string) (string, error) {
	rel, err := filepath.Rel(directory, path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve template path: %w", err)
	}
	return strings.TrimSuffix(filepath.ToSlash(rel), extension), nil
}

// IsWord reports whether s consists solely of word characters, the regular
// expression \w class of [A-Za-z0-9_]. The empty string qualifies. Engines use
// it to reject binding keys that cannot be addressed as template identifiers.
//
// The scan runs on the SIMD kernels of gofiber/utils, which cover a 32-byte
// vector per step on amd64 and a machine word per step elsewhere, rather than
// decoding s one rune at a time.
func IsWord(s string) bool {
	return simd.MemchrNotWord(utils.UnsafeBytes(s)) < 0
}

// UnsafeString returns a string sharing the backing array of b, without the
// copy a string(b) conversion makes. It is only safe when b is never written
// to again, which holds for the freshly read template sources engines hand to
// their parsers.
func UnsafeString(b []byte) string {
	return utils.UnsafeString(b)
}

// UnsafeBytes returns a byte slice sharing the backing array of s, without the
// copy a []byte(s) conversion makes. The result must only ever be read, which
// holds for handing rendered output to an io.Writer: Write is not allowed to
// modify or retain the slice it is given.
func UnsafeBytes(s string) []byte {
	return utils.UnsafeBytes(s)
}

// ReadFile reads a file from the file system or http.FileSystem.
// This wrapper provides a centralized abstraction point for file operations,
// allowing template engines to depend only on the core package while the core
// manages the underlying utils dependency.
func ReadFile(path string, fs http.FileSystem) ([]byte, error) {
	buf, err := utils.ReadFile(path, fs)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return buf, nil
}

// Walk walks the file tree rooted at directory, calling walkFn for each file or
// directory in the tree, including directory.
// This wrapper provides a centralized abstraction point for filesystem traversal,
// allowing template engines to depend only on the core package while the core
// manages the underlying utils dependency.
func Walk(fs http.FileSystem, directory string, walkFn func(path string, info os.FileInfo, err error) error) error {
	if err := utils.Walk(fs, directory, walkFn); err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}
	return nil
}
