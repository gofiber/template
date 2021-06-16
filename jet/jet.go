package jet

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/loaders/httpfs"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/utils"
)

// Engine struct
type Engine struct {
	// delimiters
	left  string
	right string
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
	// lock for funcmap and templates
	mutex sync.RWMutex
	// template funcmap
	funcmap map[string]interface{}
	// templates
	Templates *jet.Set
}

// New returns a Jet render engine for Fiber
func New(directory, extension string) *Engine {
	// jet library does not export or give us any option to modify the file extension
	if extension != ".html.jet" && extension != ".jet.html" && extension != ".jet" {
		log.Fatalf("%s extension is not a valid jet engine ['.html.jet', .jet.html', '.jet']", extension)
	}

	engine := &Engine{
		directory: directory,
		extension: extension,
		layout:    "embed",
		funcmap:   make(map[string]interface{}),
	}

	return engine
}

func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	// jet library does not export or give us any option to modify the file extension
	if extension != ".html.jet" && extension != ".jet.html" && extension != ".jet" {
		log.Fatalf("%s extension is not a valid jet engine ['.html.jet', .jet.html', '.jet']", extension)
	}

	engine := &Engine{
		directory:  "/",
		fileSystem: fs,
		extension:  extension,
		layout:     "embed",
		funcmap:    make(map[string]interface{}),
	}

	return engine
}

// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) *Engine {
	e.layout = key
	return e
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {
	e.left, e.right = left, right
	return e
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.mutex.Lock()
	e.funcmap[name] = fn
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

// Parse is deprecated, please use Load() instead
func (e *Engine) Parse() error {
	fmt.Println("Parse() is deprecated, please use Load() instead.")
	return e.Load()
}

// Parse parses the templates to the engine.
func (e *Engine) Load() (err error) {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// parse templates
	// e.Templates = jet.NewHTMLSet(e.directory)

	var loader jet.Loader

	if e.fileSystem != nil {
		loader, err = httpfs.NewLoader(e.fileSystem)

		if err != nil {
			return
		}
	} else {
		loader = jet.NewInMemLoader()
	}
	if e.debug {
		e.Templates = jet.NewSet(
			loader,
			jet.WithDelims(e.left, e.right),
			jet.InDevelopmentMode(),
		)
	} else {
		e.Templates = jet.NewSet(
			loader,
			jet.WithDelims(e.left, e.right),
		)
	}

	for name, fn := range e.funcmap {
		e.Templates.AddGlobal(name, fn)
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		l := loader.(*jet.InMemLoader)
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
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.directory, path)
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(rel, e.extension)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.fileSystem)
		if err != nil {
			return err
		}

		l.Set(name, string(buf))
		// Debugging
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}

		return err
	}

	e.loaded = true

	if _, ok := loader.(*jet.InMemLoader); ok {
		return filepath.Walk(e.directory, walkFn)
	}

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
	tmpl, err := e.Templates.GetTemplate(template)
	if err != nil || tmpl == nil {
		return fmt.Errorf("render: template %s could not be loaded: %v", template, err)
	}
	bind := jetVarMap(binding)
	if len(layout) > 0 {
		lay, err := e.Templates.GetTemplate(layout[0])
		if err != nil {
			return err
		}
		bind.Set(e.layout, func() {
			_ = tmpl.Execute(out, bind, nil)
		})
		return lay.Execute(out, bind, nil)
	}
	return tmpl.Execute(out, bind, nil)
}

func jetVarMap(binding interface{}) jet.VarMap {
	var bind jet.VarMap
	if binding == nil {
		return bind
	}
	if binds, ok := binding.(map[string]interface{}); ok {
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	} else if binds, ok := binding.(fiber.Map); ok {
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	} else if binds, ok := binding.(jet.VarMap); ok {
		bind = binds
	}
	return bind
}
