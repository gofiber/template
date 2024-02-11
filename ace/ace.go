package ace

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	core "github.com/gofiber/template"
	"github.com/gofiber/utils"
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
	engine.AddFunc(engine.LayoutName, func() error {
		return fmt.Errorf("content called unexpectedly")
	})
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
	engine.AddFunc(engine.LayoutName, func() error {
		return fmt.Errorf("content called unexpectedly")
	})
	return engine
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
		if len(e.Extension) >= len(path) || path[len(path)-len(e.Extension):] != e.Extension {
			return nil
		}
		// Get the relative file path
		// ./views/html/index.tmpl -> index.tmpl
		rel, err := filepath.Rel(e.Directory, path)
		if err != nil {
			return err
		}
		// Reverse slashes '\' -> '/' and
		// partials\footer.tmpl -> partials/footer.tmpl
		name := filepath.ToSlash(rel)
		// Remove ext from name 'index.tmpl' -> 'index'
		name = strings.TrimSuffix(name, e.Extension)
		// name = strings.Replace(name, e.Extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := utils.ReadFile(path, e.FileSystem)
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

		if e.Verbose() {
			log.Printf("views: parsed template: %s\n", name)
		}
		return err
	}

	// notify Engine that we parsed all templates
	e.SetLoaded(true)

	if e.FileSystem != nil {
		return utils.Walk(e.FileSystem, e.Directory, walkFn)
	}
	return filepath.Walk(e.Directory, walkFn)
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if !e.Loaded() || e.ShouldReload() {
		if e.ShouldReload() {
			e.LockAndSetLoaded(false)
		}

		ace.FlushCache()
		if err := e.Load(); err != nil {
			return err
		}
	}

	// Acquire read lock for accessing the template
	e.Mutex.RLock()
	tmpl := e.Templates.Lookup(name)
	e.Mutex.RUnlock()

	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	// Lock while executing layout
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	// Handle layout if specified
	if len(layout) > 0 && layout[0] != "" {
		lay := e.Templates.Lookup(layout[0])
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}

		lay.Funcs(map[string]interface{}{
			e.LayoutName: func() error {
				return tmpl.Execute(out, binding)
			},
		})
		return lay.Execute(out, binding)
	}

	return tmpl.Execute(out, binding)
}
