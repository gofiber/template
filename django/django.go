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
	"github.com/gofiber/utils/v2"
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

	// A load that failed leaves Loaded unset, so the next render retries.
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
// The result may alias binding: pongo2 copies the context before executing and
// never writes back, so only a caller that writes into it needs a map of its own.
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

// sanitizePongoContext drops the keys pongo2 cannot address as identifiers,
// handing data straight back when they are all valid - see getPongoBinding.
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

// copyValidPongoKeys is sanitizePongoContext's slow path, for data with a key
// that has to be dropped.
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
func isValidKey(key string) bool {
	for _, ch := range key {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}

// SetAutoEscape sets the auto-escape property of the template engine
func (e *Engine) SetAutoEscape(autoEscape bool) {
	// Load reads the flag under the same lock.
	e.Mutex.Lock()
	e.autoEscape = autoEscape
	e.Mutex.Unlock()
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

	// A parsed pongo2 template is immutable at execution - unless its exported
	// trim options are set, which make pongo2 rewrite the token stream in
	// place. Those renders take the write lock; the rest share the read lock,
	// held across the render since Load writes LayoutName and pongo2's
	// package-level autoescape flag that the render reads.
	e.Mutex.RLock()
	tmpl, lay, err := e.lookup(name, layout, hasLayout)
	if err != nil {
		e.Mutex.RUnlock()
		return err
	}
	if executionMutates(tmpl) || (hasLayout && executionMutates(lay)) {
		e.Mutex.RUnlock()
		e.Mutex.Lock()
		defer e.Mutex.Unlock()
		// The templates may have been reloaded while no lock was held.
		if tmpl, lay, err = e.lookup(name, layout, hasLayout); err != nil {
			return err
		}
	} else {
		defer e.Mutex.RUnlock()
	}

	bind := getPongoBinding(binding)

	if !hasLayout {
		// Same all-or-nothing behavior as Execute - pongo2 buffers internally -
		// without its two full-page copies, into a string and back to a []byte.
		return tmpl.ExecuteWriter(bind, out)
	}

	parsed, perr := tmpl.ExecuteBytes(bind)
	if perr != nil {
		return perr
	}

	// bind may alias the caller's map, so the embed key goes into our own.
	layoutBind := make(pongo2.Context, len(bind)+1)
	maps.Copy(layoutBind, bind)

	// Workaround for custom {{embed}} tag
	// Mark the `embed` variable as safe
	// it has already been escaped above
	// e.LayoutName will be 'embed'
	layoutBind[e.LayoutName] = pongo2.AsSafeValue(utils.UnsafeString(parsed))

	return lay.ExecuteWriter(layoutBind, out)
}

// lookup resolves the page and, when hasLayout, the layout template. The
// caller holds e.Mutex in either mode.
func (e *Engine) lookup(name string, layout []string, hasLayout bool) (*pongo2.Template, *pongo2.Template, error) {
	tmpl, ok := e.Templates[name]
	if !ok {
		return nil, nil, fmt.Errorf("template %s does not exist", name)
	}

	var lay *pongo2.Template
	if hasLayout {
		if lay, ok = e.Templates[layout[0]]; !ok {
			return nil, nil, fmt.Errorf("LayoutName %s does not exist", layout[0])
		}
	}
	return tmpl, lay, nil
}

// executionMutates reports whether executing tmpl writes shared template
// state: pongo2 trims the token stream in place under either trim option.
func executionMutates(tmpl *pongo2.Template) bool {
	return tmpl.Options != nil && (tmpl.Options.TrimBlocks || tmpl.Options.LStripBlocks)
}
