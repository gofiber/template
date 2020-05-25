package ace

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yosssi/ace"
)

// Engine struct
type Engine struct {
	directory string
	extension string
	funcs     map[string]interface{}

	Templates map[string]*template.Template
}

// New returns a Ace render engine for Fiber
func New(directory, extension string, funcmap ...map[string]interface{}) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,
		funcs:     make(map[string]interface{}),
		Templates: make(map[string]*template.Template),
	}
	if len(funcmap) > 0 {
		engine.funcs = funcmap[0]
	}
	if err := engine.load(); err != nil {
		log.Fatalf("ace.New(): %v", err)
	}
	return engine
}

// load parses the templates to the engine.
func (e *Engine) load() error {
	// Loop trough each directory and register template files
	err := filepath.Walk(e.directory, func(path string, info os.FileInfo, err error) error {
		path = strings.TrimRight(path, ".")
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
		name = strings.ReplaceAll(name, e.extension, "")
		// Currently ACE has no partial include support
		tmpl, err := ace.Load(strings.ReplaceAll(path, e.extension, ""), "", &ace.Options{
			Extension: e.extension[1:],
			FuncMap:   e.funcs,
		})
		if err != nil {
			return err
		}
		e.Templates[name] = tmpl
		// Debugging
		fmt.Printf("[Engine] Registered view: %s\n", path)
		return err
	})
	return err
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}) error {
	tmpl, ok := e.Templates[name]
	if !ok {
		return fmt.Errorf("Template %s does not exist", name)
	}
	return tmpl.Execute(out, binding)
}
