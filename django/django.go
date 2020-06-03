package django

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2"
)

// Engine struct
type Engine struct {
	directory string
	extension string

	Templates map[string]*pongo2.Template
}

// New returns a Handlebar render engine for Fiber
func New(directory, extension string, funcmap ...map[string]interface{}) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,

		Templates: make(map[string]*pongo2.Template),
	}
	if len(funcmap) > 0 {
		// pongo2.RegisterFilter()
	}
	if err := engine.Parse(); err != nil {
		log.Fatalf("django.New(): %v", err)
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
		tmpl, err := pongo2.FromBytes(buf)
		if err != nil {
			return err
		}
		e.Templates[name] = tmpl
		// Debugging
		//fmt.Printf("[Engine] Registered view: %s\n", name)
		return err
	})
	return err
}

func getPongoBinding(binding interface{}) pongo2.Context {
	if binding == nil {
		return nil
	}
	if binds, ok := binding.(pongo2.Context); ok {
		return binds
	}
	return binding.(map[string]interface{})
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}) error {
	tmpl, ok := e.Templates[name]
	if !ok {
		return fmt.Errorf("Template %s does not exist", name)
	}
	return tmpl.ExecuteWriter(getPongoBinding(binding), out)
}
