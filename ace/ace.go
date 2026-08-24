package ace

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	core "github.com/gofiber/template/v2"
	"github.com/yosssi/ace"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates *template.Template
	// pristine is a never-executed copy of Templates made by Load - a set
	// that has executed cannot be cloned. pool recycles the clones between
	// layout renders, so steady-state renders skip the clone and re-escape.
	pristine *template.Template
	pool     *sync.Pool
}

// New returns an Ace render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Left:       "{{",
			Right:      "}}",
			Directory:  directory,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	engine.AddFunc(engine.LayoutName, layoutUnexpected)
	return engine
}

// NewFileSystem returns an Ace render engine for Fiber with file system
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Left:       "{{",
			Right:      "}}",
			Directory:  "/",
			FileSystem: fs,
			Extension:  extension,
			LayoutName: "embed",
			Funcmap:    make(map[string]interface{}),
		},
	}
	engine.AddFunc(engine.LayoutName, layoutUnexpected)
	return engine
}

// layoutUnexpected is the layout function outside a layout render. The unused
// string result makes html/template treat the returned error as the call
// failing, rather than as a value to print into the page.
func layoutUnexpected() (string, error) {
	return "", errors.New("content called unexpectedly")
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Templates = template.New(e.Directory)

	e.Templates.Delims(e.Left, e.Right)
	e.Templates.Funcs(e.Funcmap)

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
		// Skip file if it does not equal the given template extension
		if !core.HasExtension(path, e.Extension) {
			return nil
		}
		// ./views/html/index.tmpl -> index
		name, err := core.TemplateName(e.Directory, path, e.Extension)
		if err != nil {
			return err
		}
		// Read the file
		// #gosec G304
		buf, err := core.ReadFile(path, e.FileSystem)
		if err != nil {
			return err
		}
		baseFile := name + ".ace"
		base := ace.NewFile(baseFile, buf)
		inner := ace.NewFile("", []byte{})
		src := ace.NewSource(base, inner, []*ace.File{})
		rslt, err := ace.ParseSource(src, nil)
		if err != nil {
			return err
		}
		atmpl, err := ace.CompileResult(name, rslt, &ace.Options{
			Extension:  e.Extension[1:],
			FuncMap:    e.Funcmap,
			DelimLeft:  e.Left,
			DelimRight: e.Right,
		})
		if err != nil {
			return err
		}
		_, err = e.Templates.New(name).Parse(atmpl.Lookup(name).Tree.Root.String())
		if err != nil {
			return err
		}

		if e.Verbose {
			log.Printf("views: parsed template: %s\n", name)
		}
		return err
	}

	var err error
	if e.FileSystem != nil {
		err = core.Walk(e.FileSystem, e.Directory, walkFn)
	} else {
		err = filepath.Walk(e.Directory, walkFn)
	}
	if err != nil {
		return err
	}
	if e.pristine, err = e.Templates.Clone(); err != nil {
		return err
	}
	e.pool = &sync.Pool{}

	// A load that failed leaves Loaded unset, so the next render retries.
	e.Loaded = true
	return nil
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		ace.FlushCache()
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Load replaces both sets wholesale, so a render works on the snapshot it
	// takes here and holds no lock while executing - templates are immutable
	// once loaded, and a template function may itself call Render again.
	e.Mutex.RLock()
	templates, pristine, pool := e.Templates, e.pristine, e.pool
	e.Mutex.RUnlock()

	if len(layout) > 0 && layout[0] != "" {
		// The layout function is a closure over this render's writer, so it
		// goes into a private clone of the pristine set - never into the set
		// plain renders execute.
		if pristine == nil {
			pristine = templates
		}
		// A pooled set is only ever executed after this render installs its
		// own layout closure, so nothing stale in it is reachable.
		var set *template.Template
		if pool != nil {
			if pooled, ok := pool.Get().(*template.Template); ok {
				set = pooled
			}
		}
		if set == nil {
			var cerr error
			if set, cerr = pristine.Clone(); cerr != nil {
				return fmt.Errorf("render: %w", cerr)
			}
		}
		if pool != nil {
			defer pool.Put(set)
		}

		tmpl := set.Lookup(name)
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}

		lay := set.Lookup(layout[0])
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}

		// A page holding the layout action would re-enter this closure through
		// the clone's shared func map and recurse without end.
		var embedded bool
		set.Funcs(map[string]interface{}{
			e.LayoutName: func() (string, error) {
				if embedded {
					return "", errors.New("content embedded recursively")
				}
				embedded = true
				err := tmpl.Execute(out, binding)
				embedded = false
				return "", err
			},
		})
		return lay.Execute(out, binding)
	}

	tmpl := templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	return tmpl.Execute(out, binding)
}
