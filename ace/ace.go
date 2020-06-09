package ace

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yosssi/ace"
)

// Engine struct
type Engine struct {
	// delimiters
	left  string
	right string
	// views folder
	directory string
	// views extension
	extension string
	// reload on each render
	reload bool
	// debug prints the parsed templates
	debug bool
	// lock for funcmap and templates
	mutex sync.RWMutex
	// template funcmap
	funcmap map[string]interface{}
	// templates
	Templates map[string]*template.Template
}

// New returns a Ace render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		left:      "{{",
		right:     "}}",
		directory: directory,
		extension: extension,
		funcmap:   make(map[string]interface{}),
		Templates: make(map[string]*template.Template),
	}
	engine.AddFunc("yield", func() error {
		return fmt.Errorf("yield called unexpectedly.")
	})
	return engine
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {
	e.left, e.right = left, right
	return e
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions
func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.mutex.Lock()
	e.funcmap[name] = fn
	e.mutex.Unlock()
	return e
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you don't want to restart
// the application when you edit a template file.
func (e *Engine) Reload(enabled bool) *Engine {
	e.reload = enabled
	return e
}

// Debug will print the parsed templates when Load is triggered.
func (e *Engine) Debug(enabled bool) *Engine {
	e.debug = enabled
	return e
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

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
		name = strings.Replace(name, e.extension, "", -1)
		// Currently ACE has no partial include support
		tmpl, err := ace.Load(strings.Replace(path, e.extension, "", -1), "", &ace.Options{
			Extension:  e.extension[1:],
			FuncMap:    e.funcmap,
			DelimLeft:  e.left,
			DelimRight: e.right,
		})
		if err != nil {
			return err
		}
		e.Templates[name] = tmpl
		// Debugging
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	})
	return err
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layouts ...string) error {
	// reload the views
	if e.reload {
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl, ok := e.Templates[template]
	if !ok {
		return fmt.Errorf("Template %s does not exist", template)
	}
	return tmpl.Execute(out, binding)
}
