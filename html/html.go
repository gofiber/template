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
	// Add a default function that throws an error if called unexpectedly.
	// This can be useful for debugging or ensuring certain functions are used correctly.
	engine.AddFunc(engine.LayoutName, func() error {
		return errors.New("layoutName called unexpectedly")
	})
	return engine
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
		// This enable use to invoke other templates {{ template .. }}
		// The parser keeps the source around, and buf is never written to
		// again, so hand it over without copying it into a string.
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
	// map and the nested render closures entirely and execute the template
	// straight away. The render must still not overlap a layout render, which
	// installs a closure into the func map every template in the set shares:
	// a template containing the layout action would pick it up and write into
	// the other render's writer. The shared lock lets plain renders run
	// together while excluding the layout path.
	if len(layout) == 0 || layout[0] == "" {
		e.Mutex.RLock()
		defer e.Mutex.RUnlock()

		tmpl := e.Templates.Lookup(name)
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}
		return tmpl.Execute(out, binding)
	}

	// Injecting the layout function mutates the func map shared by every
	// template in the set, so layout renders run one at a time. Look the
	// templates up inside that same critical section: taking the shared lock
	// first only to hand it straight back makes every render alternate between
	// reader and writer on the same mutex, which convoys hard under load.
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

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

// renderFuncCreate builds the closure that renders tmpl with childRenderFunc
// installed as the layout function. The innermost template is given a nil
// child: overwriting the entry matters even there, because the func map is
// shared by every template in the set, so leaving the previous render's
// closure in place would let a template containing the layout action call it -
// re-rendering into a writer that request has already finished with, or
// recursing into itself without end.
func renderFuncCreate(e *Engine, out io.Writer, binding interface{}, tmpl *template.Template, childRenderFunc func() error) func() error {
	return func() error {
		tmpl.Funcs(map[string]interface{}{
			e.LayoutName: childRenderFunc,
		})
		return tmpl.Execute(out, binding)
	}
}
