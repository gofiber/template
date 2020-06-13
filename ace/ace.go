package ace

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/template/utils"
	"github.com/yosssi/ace"
)

// Engine struct
type Engine struct {
	// delimiters
	left  string
	right string
	// views folder
	directory string
	// http.FileSystem supports embedded files
	fileSystem http.FileSystem
	// views extension
	extension string
	// layout variable name that incapsulates the template
	layout string
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
		layout:    "embed",
		funcmap:   make(map[string]interface{}),
		Templates: make(map[string]*template.Template),
	}
	engine.AddFunc(engine.layout, func() error {
		return fmt.Errorf("content called unexpectedly.")
	})
	return engine
}

func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		left:       "{{",
		right:      "}}",
		directory:  "/",
		fileSystem: fs,
		extension:  extension,
		layout:     "embed",
		funcmap:    make(map[string]interface{}),
		Templates:  make(map[string]*template.Template),
	}
	engine.AddFunc(engine.layout, func() error {
		return fmt.Errorf("content called unexpectedly.")
	})
	return engine
}

// Layout defines the variable name that will incapsulate the template
func (e *Engine) Layout(key string) *Engine {
	e.layout = key
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

	// Loop trough each directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
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
		// // Read the file
		// // #gosec G304
		// buf, err := utils.ReadFile(path, e.fileSystem)
		// if err != nil {
		// 	return err
		// }

		// file := ace.NewFile(name, buf)
		// opt := &ace.Options{
		// 	Extension:  e.extension[1:],
		// 	FuncMap:    e.funcmap,
		// 	DelimLeft:  e.left,
		// 	DelimRight: e.right,
		// }
		// source := ace.NewSource(file, nil, nil)
		// result, err := ace.ParseSource(source, opt)
		// if err != nil {
		// 	return err
		// }
		// tmpl, err := ace.CompileResult(name, result, opt)
		// if err != nil {
		// 	return err
		// }
		opt := &ace.Options{
			// .ace -> ace
			Extension:  e.extension[1:],
			FuncMap:    e.funcmap,
			DelimLeft:  e.left,
			DelimRight: e.right,
		}
		if e.fileSystem != nil {
			opt.Asset = func(p string) ([]byte, error) {
				fmt.Println("before: \t", p)

				// \errors\404.ace -> /errors/404.ace
				p = filepath.ToSlash(p)

				fmt.Println("after:  \t", p)

				return utils.ReadFile(p, e.fileSystem)
			}
		}
		tmpl, err := ace.Load(strings.Replace(path, e.extension, "", -1), "", opt)
		if err != nil {
			return err
		}
		e.Templates[name] = tmpl

		if e.debug {
			fmt.Printf("views: parsed template: %s\n", name)
		}
		return err
	}
	if e.fileSystem != nil {
		return utils.Walk(e.fileSystem, e.directory, walkFn)
	}
	return filepath.Walk(e.directory, walkFn)
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	// reload the views
	if e.reload {
		ace.FlushCache()
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl := e.Templates[template]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", template)
	}
	// TODO: layout does not work
	if len(layout) > 0 {
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}
		lay.Funcs(map[string]interface{}{
			e.layout: func() error {
				return tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}
	return tmpl.Execute(out, binding)
}
