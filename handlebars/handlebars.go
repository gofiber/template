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
	"github.com/gofiber/utils"
)

// Engine struct
type Engine struct {
	// views folder
	directory string
	// http.FileSystem supports embedded files
	fileSystem http.FileSystem
	// views extension
	extension string
	// layout variable name that incapsulates the template
	layout string
	// determines if the engine parsed all templates
	loaded bool
	// reload on each render
	reload bool
	// debug prints the parsed templates
	debug bool
	// lock for funcMap and templates
	mutex sync.RWMutex
	// object to bind custom helpers once
	registerHelpersOnce sync.Once
	// template funcMap
	funcMap map[string]interface{}
	// templates
	Templates map[string]*raymond.Template
}

// New returns a Handlebar render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,
		layout:    "embed",
		funcMap:   make(map[string]interface{}),
	}
	return engine
}

func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		directory:  "/",
		fileSystem: fs,
		extension:  extension,
		layout:     "embed",
		funcMap:    make(map[string]interface{}),
	}
	return engine
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {
	fmt.Println("delims: this method is not supported for handlebars")
	return e
}

// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) *Engine {
	e.layout = key
	return e
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.mutex.Lock()
	e.funcMap[name] = fn
	e.mutex.Unlock()
	return e
}

// AddFuncMap adds the functions from a map to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFuncMap(m map[string]interface{}) *Engine {
	e.mutex.Lock()
	for name, fn := range m {
		e.funcMap[name] = fn
	}
	e.mutex.Unlock()
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) *Engine {
	e.reload = enabled
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) *Engine {
	e.debug = enabled
	return e
}

// Parse parses the templates to the engine.
func (e *Engine) Load() (err error) {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()
	// Set template settings
	e.Templates = make(map[string]*raymond.Template)
	e.registerHelpersOnce.Do(func() {
		raymond.RegisterHelpers(e.funcMap)
	})
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
		if len(e.extension) >= len(path) || path[len(path)-len(e.extension):] != e.extension {
			return nil
		}
		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.directory, path)
		if err != nil {
			return err
		}
		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.extension)
		// name = strings.Replace(name, e.extension, "", -1)

		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.fileSystem)
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
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	if e.fileSystem != nil {
		err = utils.Walk(e.fileSystem, e.directory, walkFn)
	} else {
		err = filepath.Walk(e.directory, walkFn)
	}
	// Link templates with eachother
	for i := range e.Templates {
		for n, t := range e.Templates {
			e.Templates[i].RegisterPartialTemplate(n, t)
		}
	}
	// notify engine that we parsed all templates
	e.loaded = true
	return
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	if !e.loaded || e.reload {
		if e.reload {
			e.loaded = false
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
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}
		var bind map[string]interface{}
		if m, ok := binding.(fiber.Map); ok {
			bind = m
		} else if m, ok := binding.(map[string]interface{}); ok {
			bind = m
		} else {
			bind = make(map[string]interface{}, 1)
		}
		bind[e.layout] = raymond.SafeString(parsed)
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

// FuncMap returns the template's function map.
func (e *Engine) FuncMap() map[string]interface{} {
	return e.funcMap
}
