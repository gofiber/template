package html

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	// template options
	options []string
	// template funcmap
	funcmap map[string]interface{}

	Templates *template.Template
}

// New returns a HTML render engine for Fiber
func New(directory, extension string, funcmap ...map[string]interface{}) *Engine {
	e := &Engine{
		left:      "{{",
		right:     "}}",
		directory: directory,
		extension: extension,
		funcmap:   make(map[string]interface{}),
		Templates: template.New(directory),
	}
	if len(funcmap) > 0 {
		fmt.Println("funcmap argument is deprecated, please us engine.AddFunc")
		e.funcmap = funcmap[0]
	}
	e.AddFunc("yield", func() error {
		return fmt.Errorf("yield called unexpectedly.")
	})
	return e
}

// Option sets options for the template. Options are described by
// strings, either a simple string or "key=value".
func (e *Engine) Option(opt ...string) *Engine {
	e.options = append(e.options, opt...)
	return e
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (e *Engine) Delims(left, right string) *Engine {
	e.left, e.right = left, right
	return e
}

// AddFunc adds the function to the template's function map.
func (e *Engine) AddFunc(name string, fn interface{}) *Engine {
	e.funcmap[name] = fn
	return e
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// Set template settings
	e.Templates.Delims(e.left, e.right)
	e.Templates.Option(e.options...)
	e.Templates.Funcs(e.funcmap)

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
		// fmt.Printf("[Engine] Registered view: %s\n", name)
		return err
	})
	return err
}

// Render will execute the template name along with the given values.
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layouts ...string) error {
	// Render layouts
	if len(layouts) > 0 {
		for i := range layouts {
			// Find layout
			tmpl := e.Templates.Lookup(layouts[i])
			if tmpl != nil {
				// Add custom yield function to layour
				tmpl.Funcs(map[string]interface{}{
					"yield": func() error {
						return e.Templates.ExecuteTemplate(out, template, binding)
					},
				})
				// Execute layout
				if err := e.Templates.ExecuteTemplate(out, layouts[i], binding); err != nil {
					return err
				}
			}
		}
		return nil
	}
	// No layouts
	return e.Templates.ExecuteTemplate(out, template, binding)
}
