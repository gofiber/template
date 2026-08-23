package jet

import (
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"

	"github.com/CloudyKit/jet/v6"
	"github.com/CloudyKit/jet/v6/loaders/httpfs"
	core "github.com/gofiber/template/v2"
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

		// Set copies the source into its own store, so pass a view over buf
		// rather than allocating a second copy on the way in.
		l.Set(name, core.UnsafeString(buf))
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

	// A jet template is immutable once parsed and Execute is safe for
	// concurrent use, so renders take the shared lock and run alongside each
	// other.
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()

	tmpl, err := e.Templates.GetTemplate(name)
	if err != nil || tmpl == nil {
		return fmt.Errorf("render: template %s could not be Loaded: %w", name, err)
	}

	if len(layout) > 0 && layout[0] != "" {
		lay, err := e.Templates.GetTemplate(layout[0])
		if err != nil {
			return err
		}

		// The embed function goes into a VarMap of our own: writing it into
		// the caller's would leave a closure over this render's writer behind
		// in their data, and two renders sharing one VarMap would race on it.
		bind := jetVarMap(binding, 1)
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

	return tmpl.Execute(out, jetVarMap(binding, 0), nil)
}

// jetVarMap resolves binding into a jet.VarMap. A caller-supplied VarMap is
// passed through untouched unless extra room is asked for, in which case it is
// copied so the caller's map is never written to.
func jetVarMap(binding interface{}, extra int) jet.VarMap {
	if caller, ok := binding.(jet.VarMap); ok {
		if extra == 0 {
			return caller
		}
		bind := make(jet.VarMap, len(caller)+extra)
		maps.Copy(bind, caller)
		return bind
	}

	// Fill the VarMap straight from the binding: building a
	// map[string]interface{} first only to copy out of it again costs an extra
	// map allocation and a second pass on every render.
	bind := make(jet.VarMap, core.ViewContextLen(binding)+extra)
	core.RangeViewContext(binding, func(key string, value interface{}) {
		bind.Set(key, value)
	})
	return bind
}
