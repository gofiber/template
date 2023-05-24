package slim

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
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
	engine := &Engine{
		Engine: core.Engine{
			Left:       "{{",
			Right:      "}}",
			Directory:  directory,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	return engine
}

// NewFileSystem returns a Slim render engine for Fiber with file system
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Left:       "{{",
			Right:      "}}",
			Directory:  "/",
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
		name = strings.Replace(name, e.Extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.FileSystem)
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
		// Debugging
		if e.Verbose {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	// notify engine that we parsed all templates
	e.Loaded = true
	if e.FileSystem != nil {
		return utils.Walk(e.FileSystem, e.Directory, walkFn)
	}
	return filepath.Walk(e.Directory, walkFn)
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	if !e.Loaded || e.ShouldReload {
		if e.ShouldReload {
			e.Loaded = false
		}
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl := e.Templates[template]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", template)
	}
	if len(layout) > 0 && layout[0] != "" {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if err := tmpl.Execute(buf, binding); err != nil {
			return err
		}
		var bind map[string]interface{}
		if bind == nil {
			bind = make(map[string]interface{}, 1)
		} else if context, ok := binding.(map[string]interface{}); ok {
			bind = context
		} else {
			bind = make(map[string]interface{}, 1)
		}
		bind[e.LayoutName] = buf.String()
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
		}
		return lay.Execute(out, bind)
	}
	return tmpl.Execute(out, binding)
}
