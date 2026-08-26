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
	"sync"

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

	e.Loaded = false
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

	var err error
	if e.FileSystem != nil {
		err = core.Walk(e.FileSystem, e.Directory, walkFn)
	} else {
		err = filepath.Walk(e.Directory, walkFn)
	}
	if err != nil {
		return err
	}

	// A load that failed leaves Loaded unset, so the next render retries.
	e.Loaded = true
	return nil
}

// getPongoBinding creates a pongo2.Context containing
// only valid identifiers from a binding interface.
//
// It supports the following types:
// - pongo2.Context
// - map[string]interface{}
// It returns nil if the binding is not one of the supported types.
//
// The result may alias binding - pongo2 copies before executing, never writes back.
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

// sanitizePongoContext drops invalid keys, aliasing data when all are valid.
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

// copyValidPongoKeys copies data without its invalid keys.
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

// pongo2's autoescape flag is package-global and re-read at every top-level
// execution - {% include %} and {% ssi %} included - so it must hold a render's
// setting for that render's whole execution. The gate counts renders in flight:
// matching renders join without ever waiting - a nested render cannot deadlock -
// and a flip waits until the flag is idle.
var autoescape struct {
	sync.Mutex
	value, known bool
	active       int
}

var autoescapeIdle = sync.NewCond(&autoescape)

//nolint:revive // want is the value being stored, not a control switch
func acquireAutoescape(want bool) {
	autoescape.Lock()
	for autoescape.known && autoescape.value != want && autoescape.active > 0 {
		autoescapeIdle.Wait()
	}
	if !autoescape.known || autoescape.value != want {
		pongo2.SetAutoescape(want)
		autoescape.value, autoescape.known = want, true
	}
	autoescape.active++
	autoescape.Unlock()
}

func releaseAutoescape() {
	autoescape.Lock()
	autoescape.active--
	if autoescape.active == 0 {
		autoescapeIdle.Broadcast()
	}
	autoescape.Unlock()
}

// withAutoescape runs fn with pongo2's package flag set to want.
func withAutoescape(want bool, fn func() error) error {
	acquireAutoescape(want)
	defer releaseAutoescape()
	return fn()
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

	// Trim options make pongo2 rewrite the token stream at execution - those
	// renders hold the write lock. The rest execute lock-free on the snapshot,
	// so a template function may itself call Render again.
	e.Mutex.RLock()
	tmpl, err := e.lookup(name, "template")
	var lay *pongo2.Template
	if err == nil && hasLayout {
		lay, err = e.lookup(layout[0], "LayoutName")
	}
	esc := e.autoEscape
	layoutName := e.LayoutName
	locked := err == nil && (executionMutates(tmpl) || (hasLayout && executionMutates(lay)))
	e.Mutex.RUnlock()
	if err != nil {
		return err
	}
	if locked {
		e.Mutex.Lock()
		defer e.Mutex.Unlock()
		// The templates may have been reloaded while no lock was held.
		if tmpl, err = e.lookup(name, "template"); err != nil {
			return err
		}
		if hasLayout {
			if lay, err = e.lookup(layout[0], "LayoutName"); err != nil {
				return err
			}
		}
		esc = e.autoEscape
		layoutName = e.LayoutName
	}

	bind := getPongoBinding(binding)

	if !hasLayout {
		// pongo2 buffers internally, so a failed render writes nothing to out.
		return withAutoescape(esc, func() error {
			return tmpl.ExecuteWriter(bind, out)
		})
	}

	return withAutoescape(esc, func() error {
		parsed, err := tmpl.ExecuteBytes(bind)
		if err != nil {
			return err
		}

		// bind may alias the caller's map, so the embed key goes into our own.
		layoutBind := make(pongo2.Context, len(bind)+1)
		maps.Copy(layoutBind, bind)

		// Workaround for custom {{embed}} tag
		// Mark the `embed` variable as safe
		// it has already been escaped above
		// e.LayoutName will be 'embed'
		layoutBind[layoutName] = pongo2.AsSafeValue(utils.UnsafeString(parsed))

		return lay.ExecuteWriter(layoutBind, out)
	})
}

// lookup resolves name, wording the error with kind; the caller holds e.Mutex.
func (e *Engine) lookup(name, kind string) (*pongo2.Template, error) {
	tmpl, ok := e.Templates[name]
	if !ok {
		return nil, fmt.Errorf("%s %s does not exist", kind, name)
	}
	return tmpl, nil
}

// executionMutates reports whether executing tmpl rewrites its token stream.
func executionMutates(tmpl *pongo2.Template) bool {
	return tmpl.Options != nil && (tmpl.Options.TrimBlocks || tmpl.Options.LStripBlocks)
}
