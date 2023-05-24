package handlebars

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
	"github.com/gofiber/fiber/v2"
	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
)

// Engine struct
type Engine struct {
	core.Engine
	// object to bind custom helpers once
	registerHelpersOnce sync.Once
	// templates
	Templates map[string]*raymond.Template
}

// New returns a Handlebar render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Directory:  directory,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	return engine
}

// NewFileSystem returns a Handlebars render engine for Fiber with file system
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
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() (err error) {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	// Set template settings
	e.Templates = make(map[string]*raymond.Template)
	e.registerHelpersOnce.Do(func() {
		raymond.RegisterHelpers(e.Funcmap)
	})
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
		tmpl, err := raymond.Parse(string(buf))
		if err != nil {
			return err
		}
		// This will panic, see solution at the end of the function
		// raymond.RegisterPartialTemplate(name, tmpl)
		e.Templates[name] = tmpl

		// Debugging
		if e.Verbose {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	if e.FileSystem != nil {
		err = utils.Walk(e.FileSystem, e.Directory, walkFn)
	} else {
		err = filepath.Walk(e.Directory, walkFn)
	}
	// Link templates with eachother
	for j := range e.Templates {
		for n, template := range e.Templates {
			e.Templates[j].RegisterPartialTemplate(n, template)
		}
	}
	// notify Engine that we parsed all templates
	e.Loaded = true
	return
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
	parsed, err := tmpl.Exec(binding)
	if err != nil {
		return fmt.Errorf("render: %v", err)
	}
	if len(layout) > 0 && layout[0] != "" {
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
		}
		var bind map[string]interface{}
		if m, ok := binding.(fiber.Map); ok {
			bind = m
		} else if m, ok := binding.(map[string]interface{}); ok {
			bind = m
		} else {
			bind = make(map[string]interface{}, 1)
		}
		bind[e.LayoutName] = raymond.SafeString(parsed)
		parsed, err := lay.Exec(bind)
		if err != nil {
			return fmt.Errorf("render: %v", err)
		}
		if _, err = out.Write([]byte(parsed)); err != nil {
			return fmt.Errorf("render: %v", err)
		}
		return nil
	}
	if _, err = out.Write([]byte(parsed)); err != nil {
		return fmt.Errorf("render: %v", err)
	}
	return err
}
