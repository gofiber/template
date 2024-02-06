package django

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/flosch/pongo2/v6"
	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
)

// Engine struct
type Engine struct {
	core.Engine
	// forward the base path to the template Engine
	forwardPath bool
	// set auto escape globally
	autoEscape bool
	// templates
	Templates map[string]*pongo2.Template
}

// New returns a Django render engine for Fiber
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
		autoEscape: true,
	}
	return engine
}

// NewFileSystem returns a Django render engine for Fiber with file system
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
		autoEscape: true,
	}
	return engine
}

// NewPathForwardingFileSystem Passes "Directory" to the template engine where alternative functions don't.
//
//	This fixes errors during resolution of templates when "{% extends 'parent.html' %}" is used.
func NewPathForwardingFileSystem(fs http.FileSystem, directory, extension string) *Engine {
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
		autoEscape:  true,
		forwardPath: true,
	}
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Templates = make(map[string]*pongo2.Template)

	baseDir := e.Directory

	var pongoloader pongo2.TemplateLoader
	if e.FileSystem != nil {
		// ensures creation of httpFileSystemLoader only when filesystem is defined
		if e.forwardPath {
			pongoloader = pongo2.MustNewHttpFileSystemLoader(e.FileSystem, baseDir)
		} else {
			pongoloader = pongo2.MustNewHttpFileSystemLoader(e.FileSystem, "")
		}
	} else {
		pongoloader = pongo2.MustNewLocalFileSystemLoader(baseDir)
	}

	// New pongo2 defaultset
	pongoset := pongo2.NewSet("default", pongoloader)
	// Set template settings
	pongoset.Globals.Update(e.Funcmap)
	// Set autoescaping
	pongo2.SetAutoescape(e.autoEscape)

	// Loop trough each Directory and register template files
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
		tmpl, err := pongoset.FromBytes(buf)
		if err != nil {
			return err
		}
		e.Templates[name] = tmpl
		// Debugging
		if e.Verbose {
			log.Printf("views: parsed template: %s\n", name)
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

// getPongoBinding creates a pongo2.Context containing
// only valid identifiers from a binding interface.
//
// It supports the following types:
// - pongo2.Context
// - map[string]interface{}
// - fiber.Map
//
// It returns nil if the binding is not one of the supported types.
func getPongoBinding(binding interface{}) pongo2.Context {
	if binding == nil {
		return nil
	}
	var bind pongo2.Context
	switch binds := binding.(type) {
	case pongo2.Context:
		bind = binds
	case map[string]interface{}:
		bind = binds
	case fiber.Map:
		bind = make(pongo2.Context)
		for key, value := range binds {
			// only add valid keys
			if isValidKey(key) {
				bind[key] = value
			}
		}
		return bind
	}

	// Remove invalid keys
	for key := range bind {
		if !isValidKey(key) {
			delete(bind, key)
		}
	}

	return bind
}

// isValidKey checks if the key is valid
//
// Valid keys match the following regex: [a-zA-Z0-9_]+
func isValidKey(key string) bool {
	for _, ch := range key {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}

// SetAutoEscape sets the auto-escape property of the template engine
func (e *Engine) SetAutoEscape(autoEscape bool) {
	e.autoEscape = autoEscape
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	if !e.Loaded || e.ShouldReload {
		if e.ShouldReload {
			e.Loaded = false
		}
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl, ok := e.Templates[name]
	if !ok {
		return fmt.Errorf("template %s does not exist", name)
	}

	bind := getPongoBinding(binding)
	parsed, err := tmpl.Execute(bind)
	if err != nil {
		return err
	}
	if len(layout) > 0 && layout[0] != "" {
		if bind == nil {
			bind = make(map[string]interface{}, 1)
		}

		// Workaround for custom {{embed}} tag
		// Mark the `embed` variable as safe
		// it has already been escaped above
		// e.LayoutName will be 'embed'
		safeEmbed := pongo2.AsSafeValue(parsed)

		// Add the safe value to the binding map
		bind[e.LayoutName] = safeEmbed

		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("LayoutName %s does not exist", layout[0])
		}
		return lay.ExecuteWriter(bind, out)
	}
	if _, err = out.Write([]byte(parsed)); err != nil {
		return err
	}
	return nil
}
