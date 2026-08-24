package html

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	core "github.com/gofiber/template/v2"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates *template.Template
}

// New returns a HTML render engine for Fiber
func New(directory, extension string) *Engine {
	return newEngine(directory, extension, nil)
}

// NewFileSystem returns a HTML render engine for Fiber with file system
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
	engine.AddFunc(engine.LayoutName, layoutUnexpected)
	return engine
}

// layoutUnexpected is what the layout function is set to outside a layout
// render, so a finished render's closure can never be reached again.
func layoutUnexpected() error {
	return errors.New("layoutName called unexpectedly")
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	if e.Loaded {
		return nil
	}

	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	e.Templates = template.New(e.Directory)

	// Set template settings
	e.Templates.Delims(e.Left, e.Right)
	e.Templates.Funcs(e.Funcmap)

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
		// This enable use to invoke other templates {{ template .. }}
		_, err = e.Templates.New(name).Parse(core.UnsafeString(buf))
		if err != nil {
			return err
		}

		// Debugging
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

// Render will execute the template name along with the given values.
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Without a layout there is nothing to embed, so skip the per-render func
	// map and the nested closures. The shared lock lets plain renders run
	// together, but never alongside the layout path.
	if len(layout) == 0 || layout[0] == "" {
		e.Mutex.RLock()
		defer e.Mutex.RUnlock()

		tmpl := e.Templates.Lookup(name)
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}
		return tmpl.Execute(out, binding)
	}

	// The layout function goes into the func map the whole set shares, so
	// layout renders run one at a time, lookups included.
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	// A layout that never reaches the layout action leaves its closure - and
	// this render's writer with it - behind for the next render to call.
	defer e.Templates.Funcs(map[string]interface{}{e.LayoutName: layoutUnexpected})

	tmpl := e.Templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	// construct a nested render function to embed templates in layouts
	render := renderFuncCreate(e, out, binding, tmpl, nil)
	for _, layName := range layout {
		if layName == "" {
			break
		}
		lay := e.Templates.Lookup(layName)
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layName)
		}
		render = renderFuncCreate(e, out, binding, lay, render)
	}
	return render()
}

// renderFuncCreate renders tmpl with childRenderFunc installed as the layout
// function. The innermost template gets a nil child rather than no call at all,
// so a template holding the layout action cannot embed itself without end.
func renderFuncCreate(e *Engine, out io.Writer, binding interface{}, tmpl *template.Template, childRenderFunc func() error) func() error {
	return func() error {
		tmpl.Funcs(map[string]interface{}{
			e.LayoutName: childRenderFunc,
		})
		return tmpl.Execute(out, binding)
	}
}
