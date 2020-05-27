package mustache

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cbroglie/mustache"
)

// Engine struct
type Engine struct {
	directory string
	extension string

	Templates map[string]*mustache.Template
}

// New returns a Handlebar render engine for Fiber
func New(directory, extension string, funcmap ...map[string]interface{}) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,

		Templates: make(map[string]*mustache.Template),
	}
	if len(funcmap) > 0 {
		//raymond.RegisterHelpers(funcmap[0])
	}
	if err := engine.load(); err != nil {
		log.Fatalf("mustache.New(): %v", err)
	}
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) load() error {
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
		tmpl, err := mustache.ParseString(string(buf))
		//mustache.ParseStringPartials()

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

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}) error {
	tmpl, ok := e.Templates[name]
	if !ok {
		return fmt.Errorf("Template %s does not exist", name)
	}
	parsed, err := tmpl.Render(binding)
	if err != nil {
		return err
	}
	_, err = out.Write([]byte(parsed))
	return err
}
