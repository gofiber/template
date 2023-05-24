package django

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2/v6"
	"github.com/gofiber/fiber/v2"
	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
)

// Engine struct
type Engine struct {
	core.Engine
	// forward the base path to the template Engine
	forwardPath bool
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
	}
	return engine
}

// NewPathForwardingFileSystem Passes "Directory" to the template engine where alternative functions don't.
//
//	This fixes errors during resolution of templates when "{% extends 'parent.html' %}" is used.
func NewPathForwardingFileSystem(fs http.FileSystem, directory string, extension string) *Engine {

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
	pongo2.SetAutoescape(false)

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
		tmpl, err := pongoset.FromBytes(buf)
		if err != nil {
			return err
		}
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

func getPongoBinding(binding interface{}) pongo2.Context {
	if binding == nil {
		return nil
	}
	if binds, ok := binding.(pongo2.Context); ok {
		return binds
	}
	if binds, ok := binding.(map[string]interface{}); ok {
		return binds
	}
	if binds, ok := binding.(fiber.Map); ok {
		bind := make(pongo2.Context)
		for key, value := range binds {
			bind[key] = value
		}
		return bind
	}

	return nil
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
	tmpl, ok := e.Templates[template]
	if !ok {
		return fmt.Errorf("template %s does not exist", template)
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
		bind[e.LayoutName] = parsed
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
