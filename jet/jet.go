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
	// reload the views
	if e.reload {
		if err := e.Load(); err != nil {
			return err
		}
	}
	// load main template
	tmpl, err := e.Templates.GetTemplate(template)
	if err != nil {
		return err
	}
	// if len(layout) > 0 {
	// 	lay, err := e.Templates.GetTemplate(layout[0])
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		return err
	// 	}
	// 	buf := bytebufferpool.Get()
	// 	defer bytebufferpool.Put(buf)

	// 	binds := jetVarMap(binding)
	// 	fmt.Println("binds, ", binds)
	// 	if err := tmpl.Execute(buf, binds, nil); err != nil {
	// 		fmt.Println(err)
	// 		return err
	// 	}

	// 	binds.Set("Content", html.UnescapeString(buf.String()))
	// 	return lay.Execute(out, binds, nil)
	// }
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
