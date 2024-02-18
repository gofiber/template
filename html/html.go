package html

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
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
		return fmt.Errorf("layoutName called unexpectedly")
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
		if len(e.Extension) >= len(path) || path[len(path)-len(e.Extension):] != e.Extension {
			return nil
		}

		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.Directory, path)
		if err != nil {
			return err
		}

		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.Extension)
		// name = strings.Replace(name, e.Extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.FileSystem)
		if err != nil {
			return err
		}

		// Create new template associated with the current one
		// This enable use to invoke other templates {{ template .. }}
		_, err = e.Templates.New(name).Parse(string(buf))
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
		return utils.Walk(e.FileSystem, e.Directory, walkFn)
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

	// Acquire read lock for accessing the template
	e.Mutex.RLock()
	tmpl := e.Templates.Lookup(name)
	e.Mutex.RUnlock()

	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	render := renderFuncCreate(e, out, binding, *tmpl, nil)
	if len(layout) > 0 && layout[0] != "" {
		e.Mutex.Lock()
		defer e.Mutex.Unlock()
	}

	// construct a nested render function to embed templates in layouts
	for _, layName := range layout {
		if layName == "" {
			break
		}
		lay := e.Templates.Lookup(layName)
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layName)
		}
		render = renderFuncCreate(e, out, binding, *lay, render)
	}
	return render()
}

func renderFuncCreate(e *Engine, out io.Writer, binding interface{}, tmpl template.Template, childRenderFunc func() error) func() error {
	return func() error {
		tmpl.Funcs(map[string]interface{}{
			e.LayoutName: childRenderFunc,
		})
		return tmpl.Execute(out, binding)
	}
}
