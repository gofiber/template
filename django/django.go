package django

import (
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"

	"github.com/flosch/pongo2/v6"
	core "github.com/gofiber/template/v2"
)

// Engine struct
type Engine struct {
	core.Engine
	// forward the base path to the template Engine
	forwardPath bool
	// set auto escape globally
	autoEscape bool
	// templates
	Templates map[string]*pongo2.Template
}

// This helper function is used to avoid duplication in public constructors.
func (e *Engine) initialize(directory, extension string, fs http.FileSystem) {
	e.Left = "{{"
	e.Right = "}}"
	e.Directory = directory
	e.Extension = extension
	e.LayoutName = "embed"
	e.Funcmap = make(map[string]interface{})
	e.autoEscape = true
	e.FileSystem = fs
}

// New creates a new Engine with a directory and extension.
func New(directory, extension string) *Engine {
	engine := &Engine{}
	engine.initialize(directory, extension, nil)
	return engine
}

// NewFileSystem creates a new Engine with a file system and extension.
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{}
	engine.initialize("/", extension, fs)
	return engine
}

// NewPathForwardingFileSystem creates a new Engine with path forwarding,
// using a file system, directory, and extension.
func NewPathForwardingFileSystem(fs http.FileSystem, directory, extension string) *Engine {
	engine := &Engine{forwardPath: true}
	engine.initialize(directory, extension, fs)
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Templates = make(map[string]*pongo2.Template)
	baseDir := e.Directory
	var pongoloader pongo2.TemplateLoader

	if e.FileSystem != nil {
		// ensures creation of httpFileSystemLoader only when filesystem is defined
		if e.forwardPath {
			pongoloader = pongo2.MustNewHttpFileSystemLoader(e.FileSystem, baseDir)
		} else {
			pongoloader = pongo2.MustNewHttpFileSystemLoader(e.FileSystem, "")
		}
	} else {
		pongoloader = pongo2.MustNewLocalFileSystemLoader(baseDir)
	}

	// New pongo2 defaultset
	pongoset := pongo2.NewSet("default", pongoloader)
	// Set template settings
	pongoset.Globals.Update(e.Funcmap)
	// Set autoescaping
	pongo2.SetAutoescape(e.autoEscape)

	// Loop trough each Directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}

		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}

		// Skip file if it does not equal the given template Extension
		if !core.HasExtension(path, e.Extension) {
			return nil
		}

		// Derive the template name from the path
		// ./views/html/index.tmpl -> index
		name, err := core.TemplateName(e.Directory, path, e.Extension)
		if err != nil {
			return err
		}

		// Read the file
		// #gosec G304
		buf, err := core.ReadFile(path, e.FileSystem)
		if err != nil {
			return err
		}

		// Create new template associated with the current one
		tmpl, err := pongoset.FromBytes(buf)
		if err != nil {
			return err
		}
		e.Templates[name] = tmpl

		if e.Verbose {
			log.Printf("views: parsed template: %s\n", name)
		}
		return err
	}

	// notify Engine that we parsed all templates
	e.Loaded = true

	if e.FileSystem != nil {
		return core.Walk(e.FileSystem, e.Directory, walkFn)
	}
	return filepath.Walk(e.Directory, walkFn)
}

// getPongoBinding creates a pongo2.Context containing
// only valid identifiers from a binding interface.
//
// It supports the following types:
// - pongo2.Context
// - map[string]interface{}
// It returns nil if the binding is not one of the supported types.
//
// The result may alias binding. pongo2 copies the context into a map of its own
// before executing and never writes back, so a context that is only rendered
// needs no copy of its own; a caller that writes into the context - the layout
// path adds the embed key - has to build its own map from it.
func getPongoBinding(binding interface{}) pongo2.Context {
	if binding == nil {
		return nil
	}

	switch binds := binding.(type) {
	case pongo2.Context:
		return sanitizePongoContext(binds)
	case map[string]interface{}:
		return sanitizePongoContext(binds)
	}

	value := reflect.ValueOf(binding)
	if value.Kind() != reflect.Map || value.IsNil() {
		return nil
	}

	if value.Type().Key().Kind() != reflect.String {
		return nil
	}

	bind := make(pongo2.Context, value.Len())
	for _, key := range value.MapKeys() {
		strKey := key.String()
		if !isValidKey(strKey) {
			continue
		}
		bind[strKey] = value.MapIndex(key).Interface()
	}

	return bind
}

// sanitizePongoContext drops the keys pongo2 cannot address as identifiers. It
// hands data straight back when every key is already valid, which is the usual
// case; see getPongoBinding on why that is safe to do.
func sanitizePongoContext(data map[string]interface{}) pongo2.Context {
	if len(data) == 0 {
		return make(pongo2.Context)
	}

	for key := range data {
		if !isValidKey(key) {
			return copyValidPongoKeys(data)
		}
	}
	return data
}

// copyValidPongoKeys is the slow path of sanitizePongoContext, taken only when
// data holds a key that has to be dropped.
func copyValidPongoKeys(data map[string]interface{}) pongo2.Context {
	bind := make(pongo2.Context, len(data))
	for key, value := range data {
		if !isValidKey(key) {
			continue
		}
		bind[key] = value
	}
	return bind
}

// isValidKey checks if the key is valid
//
// Valid keys match the following regex: [a-zA-Z0-9_]+
//
// The scan is delegated to the core helper, which runs on the SIMD kernels of
// gofiber/utils instead of walking the key one rune at a time.
func isValidKey(key string) bool {
	return core.IsWord(key)
}

// SetAutoEscape sets the auto-escape property of the template engine
func (e *Engine) SetAutoEscape(autoEscape bool) {
	e.autoEscape = autoEscape
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	hasLayout := len(layout) > 0 && layout[0] != ""

	// Acquire read lock for accessing the templates. Rendering only reads
	// them - a pongo2 template is immutable once parsed - so concurrent
	// renders share the lock instead of queueing behind an exclusive one.
	e.Mutex.RLock()
	tmpl, ok := e.Templates[name]
	var lay *pongo2.Template
	if hasLayout {
		lay = e.Templates[layout[0]]
	}
	e.Mutex.RUnlock()

	if !ok {
		return fmt.Errorf("template %s does not exist", name)
	}

	bind := getPongoBinding(binding)

	if !hasLayout {
		// pongo2 buffers the render internally and writes nothing on error, so
		// this keeps the all-or-nothing behaviour of Execute while dropping
		// both full-page copies it needed: one into a string, one back out
		// into a []byte for the writer.
		return tmpl.ExecuteWriter(bind, out)
	}

	parsed, err := tmpl.ExecuteBytes(bind)
	if err != nil {
		return err
	}

	if lay == nil {
		return fmt.Errorf("LayoutName %s does not exist", layout[0])
	}

	// bind may alias the caller's map, so the embed key goes into a context of
	// our own rather than into theirs.
	layoutBind := make(pongo2.Context, len(bind)+1)
	maps.Copy(layoutBind, bind)

	// Workaround for custom {{embed}} tag
	// Mark the `embed` variable as safe
	// it has already been escaped above
	// e.LayoutName will be 'embed'
	// The buffer behind parsed is never reused, so the rendered body goes in as
	// a view over it rather than a copy.
	layoutBind[e.LayoutName] = pongo2.AsSafeValue(core.UnsafeString(parsed))

	return lay.ExecuteWriter(layoutBind, out)
}
