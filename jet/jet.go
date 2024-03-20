package jet

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/loaders/httpfs"
	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates *jet.Set
}

// New returns a Jet render engine for Fiber
func New(directory, extension string) *Engine {
	// jet library does not export or give us any option to modify the file extension
	if extension != ".html.jet" && extension != ".jet.html" && extension != ".jet" {
		log.Fatalf("%s Extension is not a valid jet engine ['.html.jet', .jet.html', '.jet']", extension) //nolint:revive // this is not an issue
	}

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

// NewFileSystem returns a Jet render engine for Fiber with file system
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	// jet library does not export or give us any option to modify the file extension
	if extension != ".html.jet" && extension != ".jet.html" && extension != ".jet" {
		log.Fatalf("%s Extension is not a valid jet engine ['.html.jet', .jet.html', '.jet']", extension) //nolint:revive // this is not an issue
	}

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
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	// parse templates
	// e.Templates = jet.NewHTMLSet(e.Directory)
	var loader jet.Loader
	var err error

	if e.FileSystem != nil {
		loader, err = httpfs.NewLoader(e.FileSystem)
		if err != nil {
			return err
		}
	} else {
		loader = jet.NewInMemLoader()
	}

	if e.Verbose {
		e.Templates = jet.NewSet(
			loader,
			jet.WithDelims(e.Left, e.Right),
			jet.InDevelopmentMode(),
		)
	} else {
		e.Templates = jet.NewSet(
			loader,
			jet.WithDelims(e.Left, e.Right),
		)
	}

	for name, fn := range e.Funcmap {
		e.Templates.AddGlobal(name, fn)
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		l := loader.(*jet.InMemLoader) //nolint:errcheck,forcetypeassert // check line 106
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

		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.Directory, path)
		if err != nil {
			return err
		}

		name := strings.TrimSuffix(rel, e.Extension)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.FileSystem)
		if err != nil {
			return err
		}

		l.Set(name, string(buf))
		if e.Verbose {
			log.Printf("views: parsed template: %s\n", name)
		}

		return err
	}

	// notify Engine that we parsed all templates
	e.Loaded = true

	if _, ok := loader.(*jet.InMemLoader); ok {
		return filepath.Walk(e.Directory, walkFn)
	}

	return err
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Acquire read lock for accessing the template
	e.Mutex.RLock()
	tmpl, err := e.Templates.GetTemplate(name)
	e.Mutex.RUnlock()

	if err != nil || tmpl == nil {
		return fmt.Errorf("render: template %s could not be Loaded: %w", name, err)
	}

	// Lock while executing layout
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	bind := jetVarMap(binding)

	if len(layout) > 0 && layout[0] != "" {
		lay, err := e.Templates.GetTemplate(layout[0])
		if err != nil {
			return err
		}
		var renderingError error
		bind.Set(e.LayoutName, func() {
			renderingError = tmpl.Execute(out, bind, nil)
		})
		err = lay.Execute(out, bind, nil)
		if renderingError != nil {
			return renderingError
		}
		return err
	}
	return tmpl.Execute(out, bind, nil)
}

func jetVarMap(binding interface{}) jet.VarMap {
	var bind jet.VarMap
	if binding == nil {
		return bind
	}
	switch binds := binding.(type) {
	case map[string]interface{}:
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	case fiber.Map:
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	case jet.VarMap:
		bind = binds
	}
	return bind
}
