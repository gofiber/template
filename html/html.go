package html

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Engine struct
type Engine struct {
	directory string
	extension string

	Templates *template.Template
}

// New returns a HTML render engine for Fiber
func New(directory, extension string, funcmap ...map[string]interface{}) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,
		Templates: template.New(directory),
	}
	if len(funcmap) > 0 {
		engine.Templates.Funcs(funcmap[0])
	}
	funcs := template.FuncMap{
		"yield": func() error {
			return fmt.Errorf("yield called unexpectedly.")
		},
	}
	engine.Templates.Funcs(funcs)

	if err := engine.Parse(); err != nil {
		log.Fatalf("html.New(): %v", err)
	}
	return engine
}

// Parse parses the templates to the engine.
func (e *Engine) Parse() error {
	// Loop trough each directory and register template files
	err := filepath.Walk(e.directory, func(path string, info os.FileInfo, err error) error {
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
		if ext != e.extension {
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
		name = strings.Replace(name, e.extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		// Create new template associated with the current one
		// This enable use to invoke other templates {{ template .. }}
		_, err = e.Templates.New(name).Parse(string(buf))
		if err != nil {
			return err
		}
		// Debugging
		//fmt.Printf("[Engine] Registered view: %s\n", name)
		return err
	})
	return err
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	if len(layout) > 0 {
		tmpl := e.Templates.Lookup(layout[0])
		tmpl.Funcs(template.FuncMap{
			"yield": func() error {
				return e.Templates.ExecuteTemplate(out, name, binding)
			},
		})
		return e.Templates.ExecuteTemplate(out, layout[0], binding)
	}
	return e.Templates.ExecuteTemplate(out, name, binding)
}
