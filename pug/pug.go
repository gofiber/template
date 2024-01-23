package pug

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Joker/hpp"
	"github.com/Joker/jade"
	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
)

// Engine struct
type Engine struct {
	core.Engine
	//  templates
	Templates *template.Template
}

// New returns a Pug render engine for Fiber
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
	engine.AddFunc(engine.LayoutName, func() error {
		return fmt.Errorf("layoutName called unexpectedly")
	})
	return engine
}

// NewFileSystem returns a Pug render engine for Fiber with file system
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
	engine.AddFunc(engine.LayoutName, func() error {
		return fmt.Errorf("layoutName called unexpectedly")
	})
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
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
		// Get file extension of file
		ext := filepath.Ext(path)
		// Skip file if it does not equal the given template extension
		if ext != e.Extension {
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
		// name = strings.Replace(name, e.extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.FileSystem)
		if err != nil {
			return err
		}
		// Create new template associated with the current one
		// This enable use to invoke other templates {{ template .. }}
		var pug string
		if e.FileSystem != nil {
			pug, err = jade.ParseWithFileSystem(path, buf, e.FileSystem)
		} else {
			pug, err = jade.Parse(path, buf)
		}
		if err != nil {
			return err
		}
		_, err = e.Templates.New(name).Parse(hpp.PrPrint(pug))
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

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	e.Mutex.Lock()
	sdefer e.Mutex.Unlock()

	if !e.Loaded || e.ShouldReload {
		if e.ShouldReload {
			e.Loaded = false
		}
		if err := e.Load(); err != nil {
			return err
		}
	}

	tmpl := e.Templates.Lookup(name)

	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	if len(layout) > 0 && layout[0] != "" {
		lay := e.Templates.Lookup(layout[0])
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}
		lay.Funcs(map[string]interface{}{
			e.LayoutName: func() error {
				return tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}
	return tmpl.Execute(out, binding)
}
