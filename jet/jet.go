package jet

import (
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/CloudyKit/jet/v3"
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
	// lock for funcmap and templates
	mutex sync.RWMutex
	// template funcmap
	funcmap map[string]interface{}
	// templates
	Templates *jet.Set
}

// New returns a Jet render engine for Fiber
func New(directory, extension string) *Engine {
	// jet library does not export or give us any option to modify the file extension
	extJet := []string{
		".html.jet",
		".jet.html",
		".jet",
	}
	extOK := false
	for _, ext := range extJet {
		if ext == extension {
			extOK = true
			break
		}
	}
	if !extOK {
		log.Fatalf("%s extension is not a valid jet engine ['.html.jet', .jet.html', '.jet']", extension)
	}

	engine := &Engine{
		directory: directory,
		extension: extension,
		funcmap:   make(map[string]interface{}),
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

// Parse parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// parse templates
	e.Templates = jet.NewHTMLSet(e.directory)
	fmt.Println(e.Templates)

	// Set template settings
	e.Templates.Delims(e.left, e.right)
	e.Templates.SetDevelopmentMode(e.reload)

	for name, fn := range e.funcmap {
		e.Templates.AddGlobal(name, fn)
	}
	return nil
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layouts ...string) error {
	tmpl, err := e.Templates.GetTemplate(template)
	if err != nil {
		return err
	}
	return tmpl.Execute(out, jetVarMap(binding), nil)
}

func jetVarMap(binding interface{}) jet.VarMap {
	var jetVarMap jet.VarMap
	if binding != nil {
		if binds, ok := binding.(jet.VarMap); ok {
			jetVarMap = binds
		} else {
			binds, ok := binding.(map[string]interface{})
			if ok {
				jetVarMap = make(jet.VarMap)
				for key, value := range binds {
					jetVarMap.Set(key, value)
				}
			}
		}
	}
	return jetVarMap
}
