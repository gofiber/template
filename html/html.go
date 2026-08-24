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
	"sync"

	core "github.com/gofiber/template/v2"
	"github.com/gofiber/utils/v2"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates *template.Template
	// pristine is a never-executed copy of Templates made by Load - a set
	// that has executed cannot be cloned. pool recycles the clones between
	// layout renders, so steady-state renders skip the clone and re-escape.
	pristine *template.Template
	pool     *sync.Pool
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

// layoutUnexpected is the layout function outside a layout render. The unused
// string result makes html/template treat the returned error as the call
// failing, rather than as a value to print into the page.
func layoutUnexpected() (string, error) {
	return "", errors.New("layoutName called unexpectedly")
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	if e.Loaded {
		return nil
	}

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
		_, err = e.Templates.New(name).Parse(utils.UnsafeString(buf))
		if err != nil {
			return err
		}

		// Debugging
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
	if e.pristine, err = e.Templates.Clone(); err != nil {
		return err
	}
	e.pool = &sync.Pool{}

	// A load that failed leaves Loaded unset, so the next render retries.
	e.Loaded = true
	return nil
}

// Render will execute the template name along with the given values.
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Load replaces both sets wholesale, so a render works on the snapshot it
	// takes here and holds no lock while executing - templates are immutable
	// once loaded, and a template function may itself call Render again.
	e.Mutex.RLock()
	templates, pristine, pool := e.Templates, e.pristine, e.pool
	e.Mutex.RUnlock()

	// Without a layout there is nothing to embed, so skip the per-render func
	// map and the nested closures.
	if len(layout) == 0 || layout[0] == "" {
		tmpl := templates.Lookup(name)
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}
		return tmpl.Execute(out, binding)
	}

	// The layout function is a closure over this render's writer, so it goes
	// into a private clone of the pristine set - never into the set plain
	// renders execute.
	if pristine == nil {
		pristine = templates
	}
	// A pooled set is only ever executed after this render installs its own
	// layout chain, so nothing stale in it is reachable.
	var set *template.Template
	if pool != nil {
		if pooled, ok := pool.Get().(*template.Template); ok {
			set = pooled
		}
	}
	if set == nil {
		var cerr error
		if set, cerr = pristine.Clone(); cerr != nil {
			return fmt.Errorf("render: %w", cerr)
		}
	}
	if pool != nil {
		defer pool.Put(set)
	}

	tmpl := set.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	// construct a nested render function to embed templates in layouts
	render := renderFuncCreate(e, out, binding, tmpl, nil)
	for _, layName := range layout {
		if layName == "" {
			break
		}
		lay := set.Lookup(layName)
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layName)
		}
		render = renderFuncCreate(e, out, binding, lay, render)
	}
	return render()
}

// renderFuncCreate renders tmpl with childRenderFunc installed as the layout
// function. The innermost template gets layoutUnexpected, so a template holding
// the layout action fails cleanly instead of embedding itself without end.
func renderFuncCreate(e *Engine, out io.Writer, binding interface{}, tmpl *template.Template, childRenderFunc func() error) func() error {
	return func() error {
		embed := interface{}(layoutUnexpected)
		if childRenderFunc != nil {
			embed = func() (string, error) { return "", childRenderFunc() }
		}
		tmpl.Funcs(map[string]interface{}{
			e.LayoutName: embed,
		})
		return tmpl.Execute(out, binding)
	}
}
