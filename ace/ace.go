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
	// pristine is never executed - an executed set cannot be cloned. pool
	// recycles the layout-render clones and their escape work.
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

// layoutUnexpected is the layout function outside a layout render - the
// (string, error) shape makes html/template fail the call instead of printing it.
func layoutUnexpected() (string, error) {
	return "", errors.New("content called unexpectedly")
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Loaded = false
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

	// Renders execute lock-free on this snapshot: the sets are immutable once
	// loaded, and a template function may itself call Render again.
	e.Mutex.RLock()
	templates, pristine, pool := e.Templates, e.pristine, e.pool
	layoutName := e.LayoutName
	e.Mutex.RUnlock()

	if len(layout) > 0 && layout[0] != "" {
		// The embed closure holds this render's writer, so it goes into a private clone.
		if pristine == nil {
			pristine = templates
		}
		// A pooled set always gets this render's closure before executing.
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
			// The closure holds this render's writer; the sentinel replaces it in the pool.
			defer func() {
				set.Funcs(map[string]interface{}{layoutName: layoutUnexpected})
				pool.Put(set)
			}()
		}

		tmpl := set.Lookup(name)
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}

		lay := set.Lookup(layout[0])
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}

		// A self-embedding page would recurse without end through the func map.
		var embedded bool
		set.Funcs(map[string]interface{}{
			layoutName: func() (string, error) {
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
