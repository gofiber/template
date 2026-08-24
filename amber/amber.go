package amber

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

	"github.com/eknkc/amber"
	core "github.com/gofiber/template/v2"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates map[string]*template.Template
	// pristine holds never-executed copies - an executed template cannot be
	// cloned. pools recycles the layout-render clones and their escape work.
	pristine map[string]*template.Template
	pools    map[string]*sync.Pool
}

// New returns an Amber render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Directory:  directory,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	engine.AddFunc(engine.LayoutName, layoutUnexpected)
	return engine
}

// NewFileSystem returns an Amber render engine for Fiber with file system
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Directory:  "/",
			FileSystem: fs,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	engine.AddFunc(engine.LayoutName, layoutUnexpected)
	return engine
}

// layoutUnexpected is the layout function outside a layout render - the
// (string, error) shape makes html/template fail the call instead of printing it.
func layoutUnexpected() (string, error) {
	return "", errors.New("layoutName called unexpectedly")
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Loaded = false
	e.Templates = make(map[string]*template.Template)

	// Set template settings
	// prepare the global amber funcs
	funcs := template.FuncMap{}

	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}

	for k, v := range e.Funcmap {
		funcs[k] = v
	}

	amber.FuncMap = funcs //nolint:reassign // this is fine, as long as it's not run in parallel in a test.

	// Loop trough each directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}
		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}
		// Skip file if it does not equal the given template extension
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
		option := amber.DefaultOptions
		if e.FileSystem != nil {
			option.VirtualFilesystem = e.FileSystem
		}
		tmpl, err := amber.CompileData(buf, name, option)
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
	e.pristine = make(map[string]*template.Template, len(e.Templates))
	e.pools = make(map[string]*sync.Pool, len(e.Templates))
	for name, tmpl := range e.Templates {
		if e.pristine[name], err = tmpl.Clone(); err != nil {
			return err
		}
		e.pools[name] = &sync.Pool{}
	}

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

	// Renders execute lock-free on this snapshot: the maps are immutable once
	// loaded, and a template function may itself call Render again.
	e.Mutex.RLock()
	templates, pristine, pools := e.Templates, e.pristine, e.pools
	e.Mutex.RUnlock()

	if len(layout) > 0 && layout[0] != "" {
		tmpl := templates[name]
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}

		// The embed closure holds this render's writer, so it goes into a private clone.
		var lay *template.Template
		pool := pools[layout[0]]
		if pool != nil {
			if pooled, ok := pool.Get().(*template.Template); ok {
				lay = pooled
			}
		}
		if lay == nil {
			pl := pristine[layout[0]]
			if pl == nil {
				pl = templates[layout[0]]
			}
			if pl == nil {
				return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
			}
			var cerr error
			if lay, cerr = pl.Clone(); cerr != nil {
				return fmt.Errorf("render: %w", cerr)
			}
		}
		if pool != nil {
			defer pool.Put(lay)
		}
		lay.Funcs(map[string]interface{}{
			e.LayoutName: func() (string, error) {
				return "", tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}

	tmpl := templates[name]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}
	return tmpl.Execute(out, binding)
}
