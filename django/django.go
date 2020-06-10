package django

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flosch/pongo2"
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
	Templates map[string]*pongo2.Template
}

// New returns a Handlebar render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		left:      "{{",
		right:     "}}",
		directory: directory,
		extension: extension,
		funcmap:   make(map[string]interface{}),
		Templates: make(map[string]*pongo2.Template),
	}
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

// Parse is deprecated, please use Load() instead
func (e *Engine) Parse() error {
	fmt.Println("Parse() is deprecated, please use Load() instead.")
	return e.Load()
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Set template settings
	loader, err := pongo2.NewLocalFileSystemLoader(filepath.Clean(e.directory))
	if err != nil {
		return err
	}
	set := pongo2.NewSet("", loader)
	set.Globals = e.funcmap
	pongo2.SetAutoescape(false)
	// Loop trough each directory and register template files
	err = filepath.Walk(e.directory, func(path string, info os.FileInfo, err error) error {
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
		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
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
	if binds, ok := binding.(map[string]interface{}); ok {
		return binds
	}
	return nil
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	// reload the views
	if e.reload {
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl, ok := e.Templates[template]
	if !ok {
		return fmt.Errorf("template %s does not exist", template)
	}
	context := getPongoBinding(binding)

	// Render layouts if provided
	if len(layout) > 0 {
		parsed, err := tmpl.Execute(getPongoBinding(binding))
		if err != nil {
			return err
		}
		if context == nil {
			context = make(map[string]interface{}, 1)
		}
		context["yield"] = parsed
		// Find layout
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("layout %s does not exist", layout[0])
		}
		return lay.ExecuteWriter(context, out)

	}
	return tmpl.ExecuteWriter(context, out)
}
