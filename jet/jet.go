package jet

import (
	"io"
	"log"

	"github.com/CloudyKit/jet"
)

// Engine struct
type Engine struct {
	directory string
	extension string

	Templates *jet.Set
}

// New returns a Jet render engine for Fiber
func New(directory, extension string, funcmap ...map[string]interface{}) *Engine {
	engine := &Engine{
		directory: directory,
		extension: extension,
	}
	if len(funcmap) > 0 {
		for key, value := range funcmap[0] {
			engine.Templates.AddGlobal(key, value)
		}
	}
	if err := engine.Parse(); err != nil {
		log.Fatalf("jet.New(): %v", err)
	}
	return engine
}

// Parse parses the templates to the engine.
func (e *Engine) Parse() error {
	e.Templates = jet.NewHTMLSet(e.directory)
	return nil
}

func getJetBinding(binding interface{}) jet.VarMap {
	if binding == nil {
		return nil
	}
	if binds, ok := binding.(jet.VarMap); ok {
		return binds
	}
	binds, ok := binding.(map[string]interface{})
	if !ok {
		return nil
	}
	varmap := make(jet.VarMap)
	for key, value := range binds {
		varmap.Set(key, value)
	}
	return varmap
}

// Execute will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}) error {
	tmpl, err := e.Templates.GetTemplate(name)
	if err != nil {
		return err
	}
	return tmpl.Execute(out, getJetBinding(binding), nil)
}
