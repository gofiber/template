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

	"github.com/eknkc/amber"
	core "github.com/gofiber/template/v2"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates map[string]*template.Template
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
	engine.AddFunc(engine.LayoutName, func() error {
		return errors.New("layoutName called unexpectedly")
	})
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
	engine.AddFunc(engine.LayoutName, func() error {
		return errors.New("layoutName called unexpectedly")
	})
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

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

	// Injecting the layout function mutates the func map shared by every
	// template in the set, so layout renders run one at a time. The lookups
	// ride along inside that same critical section: taking the shared lock
	// first only to hand it straight back makes every render alternate between
	// reader and writer on the same mutex, which convoys hard under load.
	if len(layout) > 0 && layout[0] != "" {
		e.Mutex.Lock()
		defer e.Mutex.Unlock()

		tmpl := e.Templates[name]
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}

		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
		}
		lay.Funcs(map[string]interface{}{
			e.LayoutName: func() error {
				return tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}

	// A plain render only reads, and html/template.Execute is safe for
	// concurrent use, so the shared lock only has to cover the lookup.
	e.Mutex.RLock()
	tmpl := e.Templates[name]
	e.Mutex.RUnlock()

	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}
	return tmpl.Execute(out, binding)
}
