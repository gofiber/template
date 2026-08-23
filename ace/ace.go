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

	core "github.com/gofiber/template/v2"
	"github.com/yosssi/ace"
)

// Engine struct
type Engine struct {
	core.Engine
	// templates
	Templates *template.Template
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

// layoutUnexpected is the layout function the engine keeps installed outside a
// layout render. The func map is shared by every template in the set, so the
// closure a layout render installs has to be replaced when that render ends:
// left in place, a later render of a template containing the layout action
// would call it and write into a writer that render has already finished with.
func layoutUnexpected() error {
	return errors.New("content called unexpectedly")
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
		// Derive the template name from the path
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

	// notify Engine that we parsed all templates
	e.Loaded = true

	if e.FileSystem != nil {
		return core.Walk(e.FileSystem, e.Directory, walkFn)
	}
	return filepath.Walk(e.Directory, walkFn)
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

	// Handle layout if specified. Injecting the layout function mutates the func map shared by every
	// template in the set, so layout renders run one at a time. The lookups
	// ride along inside that same critical section: taking the shared lock
	// first only to hand it straight back makes every render alternate between
	// reader and writer on the same mutex, which convoys hard under load.
	if len(layout) > 0 && layout[0] != "" {
		e.Mutex.Lock()
		defer e.Mutex.Unlock()

		tmpl := e.Templates.Lookup(name)
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}

		lay := e.Templates.Lookup(layout[0])
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}

		defer lay.Funcs(map[string]interface{}{e.LayoutName: layoutUnexpected})
		lay.Funcs(map[string]interface{}{
			e.LayoutName: func() error {
				return tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}

	// A plain render only reads, but it must not overlap a layout render: that
	// one installs a closure into the func map every template in the set
	// shares, and a template containing the layout action would pick it up and
	// write into the other render's writer. The shared lock lets plain renders
	// run together while excluding the layout path.
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()

	tmpl := e.Templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	return tmpl.Execute(out, binding)
}
