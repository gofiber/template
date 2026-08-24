package handlebars

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	core "github.com/gofiber/template/v2"
	"github.com/gofiber/utils/v2"
	"github.com/mailgun/raymond/v2"
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
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	var err error

	// Set template settings
	e.Templates = make(map[string]*raymond.Template)
	e.registerHelpersOnce.Do(func() {
		raymond.RegisterHelpers(e.Funcmap)
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
		tmpl, err := raymond.Parse(utils.UnsafeString(buf))
		if err != nil {
			return err
		}
		// This will panic, see solution at the end of the function
		// raymond.RegisterPartialTemplate(name, tmpl)
		e.Templates[name] = tmpl

		if e.Verbose {
			log.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	if e.FileSystem != nil {
		err = core.Walk(e.FileSystem, e.Directory, walkFn)
	} else {
		err = filepath.Walk(e.Directory, walkFn)
	}
	if err != nil {
		return err
	}
	// Link templates with eachother
	for j := range e.Templates {
		for n, template := range e.Templates {
			e.Templates[j].RegisterPartialTemplate(n, template)
		}
	}

	// A load that failed leaves Loaded unset, so the next render retries.
	e.Loaded = true
	return nil
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Exclusive: raymond lazily parses global source partials with no lock of its own.
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	tmpl := e.Templates[name]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	parsed, err := tmpl.Exec(binding)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	if len(layout) > 0 && layout[0] != "" {
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: LayoutName %s does not exist", layout[0])
		}
		// Our own context: the embed key must not land in the caller's map.
		bind := core.NewViewContext(binding, 1)
		bind[e.LayoutName] = raymond.SafeString(parsed)
		if parsed, err = lay.Exec(bind); err != nil {
			return fmt.Errorf("render: %w", err)
		}
	}

	if _, err = out.Write(utils.UnsafeBytes(parsed)); err != nil {
		return fmt.Errorf("render: %w", err)
	}
	return nil
}
