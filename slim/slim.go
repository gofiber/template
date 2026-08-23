package slim

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	core "github.com/gofiber/template/v2"
	"github.com/mattn/go-slim"
	"github.com/valyala/bytebufferpool"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates map[string]*slim.Template
}

type slimFunc = func(...slim.Value) (slim.Value, error)

// New returns a Slim render engine for Fiber
func New(directory, extension string) *Engine {
	return newEngine(directory, extension, nil)
}

// NewFileSystem returns a Slim render engine for Fiber with file system
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	return newEngine("/", extension, fs)
}

// newEngine creates a new Engine instance with common initialization logic.
func newEngine(directory, extension string, fs http.FileSystem) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Left:       "{{",
			Right:      "}}",
			Directory:  directory,
			FileSystem: fs,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Templates = make(map[string]*slim.Template)

	// Loop trough each Directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}
		// Skip file if it's a Directory or has no file info
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
		tmpl, err := slim.Parse(bytes.NewReader(buf))
		if err != nil {
			return err
		}

		// Init func map
		newFuncMap := make(slim.Funcs, len(e.Funcmap))
		for key, val := range e.Funcmap {
			slimFunc, ok := val.(slimFunc)
			if !ok {
				panic("slim: function must be compatible with slim.Func type. Slim does not support other types")
			}
			newFuncMap[key] = slimFunc
		}
		tmpl.FuncMap(newFuncMap)
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

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Lock while executing. go-slim's Execute registers a `render` helper that
	// writes into the template's own map of nested templates, so renders of
	// the same template cannot overlap. The lookups ride along inside that
	// same critical section: taking the shared lock first only to hand it
	// straight back makes every render alternate between reader and writer on
	// the same mutex, which convoys hard under load.
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	tmpl := e.Templates[name]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	if len(layout) > 0 && layout[0] != "" {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if err := tmpl.Execute(buf, binding); err != nil {
			return err
		}
		// The embed key goes into a context of our own: writing it into the
		// caller's map would leak the rendered body back to them, and a
		// binding that is not literally a map[string]interface{} - fiber.Map
		// is one - would otherwise reach the layout with none of its values.
		bind := core.NewViewContext(binding, 1)
		bind[e.LayoutName] = buf.String()
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
		}
		return lay.Execute(out, bind)
	}
	return tmpl.Execute(out, binding)
}
