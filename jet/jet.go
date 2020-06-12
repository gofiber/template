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
	// layout variable name that incapsulates the template
	layout string
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
		layout:    "embed",
		funcmap:   make(map[string]interface{}),
	}

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
	fmt.Println("debug: this method is not supported for jet")
	return e
}

// Parse is deprecated, please use Load() instead
func (e *Engine) Parse() error {
	fmt.Println("Parse() is deprecated, please use Load() instead.")
	return e.Load()
}

// Parse parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// parse templates
	e.Templates = jet.NewHTMLSet(e.directory)

	// Set template settings
	e.Templates.Delims(e.left, e.right)
	e.Templates.SetDevelopmentMode(false)

	qq, ok := e.Templates.LookupGlobal("yield")
	fmt.Println(qq, ok)
	for name, fn := range e.funcmap {

		e.Templates.AddGlobal(name, fn)
	}
	return nil
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, template string, binding interface{}, layout ...string) error {
	if e.reload {
		if err := e.Load(); err != nil {
			return err
		}
	}
	tmpl, err := e.Templates.GetTemplate(template)
	if err != nil || tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", template)
	}
	bind := jetVarMap(binding)
	// TODO: layout does not work
	if len(layout) > 0 {
		lay, err := e.Templates.GetTemplate(layout[0])
		if err != nil {
			return err
		}
		bind.Set(e.layout, func() {
			_ = tmpl.Execute(out, bind, nil)
		})
		return lay.Execute(out, bind, nil)
	}
	return tmpl.Execute(out, bind, nil)
}

func jetVarMap(binding interface{}) jet.VarMap {
	var bind jet.VarMap
	if binding == nil {
		return bind
	}
	if binds, ok := binding.(map[string]interface{}); ok {
		bind = make(jet.VarMap)
		for key, value := range binds {
			bind.Set(key, value)
		}
	} else if binds, ok := binding.(jet.VarMap); ok {
		bind = binds
	}
	return bind
}
