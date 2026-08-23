package mustache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cbroglie/mustache"
	core "github.com/gofiber/template/v2"
	"github.com/valyala/bytebufferpool"
)

// Engine struct
type Engine struct {
	core.Engine
	// partialsProvider supports partials for embedded files
	partialsProvider *fileSystemPartialProvider
	//  templates
	Templates map[string]*mustache.Template
}

type fileSystemPartialProvider struct {
	fileSystem http.FileSystem
	extension  string
}

func (p fileSystemPartialProvider) Get(path string) (string, error) {
	buf, err := core.ReadFile(path+p.extension, p.fileSystem)
	// The parser keeps the source around, and buf is never written to again,
	// so hand it over without copying it into a string.
	return core.UnsafeString(buf), err
}

// New returns a Mustache render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		Engine: core.Engine{
			Directory:  directory,
			Extension:  extension,
			LayoutName: "embed",
		},
	}
	return engine
}

// NewFileSystem returns a Mustache render engine for Fiber that supports embedded files
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	return NewFileSystemPartials(fs, extension, fs)
}

// NewFileSystemPartials returns a Handlebar render engine for Fiber that supports embedded files
func NewFileSystemPartials(fs http.FileSystem, extension string, partialsFS http.FileSystem) *Engine {
	engine := &Engine{
		partialsProvider: &fileSystemPartialProvider{
			fileSystem: partialsFS,
			extension:  extension,
		},
		Engine: core.Engine{
			Directory:  "/",
			FileSystem: fs,

			Extension:  extension,
			LayoutName: "embed",
		},
	}
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Templates = make(map[string]*mustache.Template)

	// Loop trough each directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
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

		// Create new template associated with the current one
		// This enable use to invoke other templates {{ template .. }}
		// The parser keeps the source around, and buf is never written to
		// again, so hand it over without copying it into a string.
		source := core.UnsafeString(buf)
		var tmpl *mustache.Template
		if e.partialsProvider != nil {
			tmpl, err = mustache.ParseStringPartials(source, e.partialsProvider)
		} else {
			tmpl, err = mustache.ParseString(source)
		}
		if err != nil {
			return err
		}

		e.Templates[name] = tmpl
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
		if err := e.Load(); err != nil {
			return err
		}
	}

	// The layout branch writes the embed key into the view context, which
	// AcquireViewContext may hand straight back from the caller, so it holds
	// the exclusive lock. The lookups ride along inside that same critical
	// section: taking the shared lock first only to hand it straight back
	// makes every render alternate between reader and writer on the same
	// mutex, which convoys hard under load.
	if len(layout) > 0 && layout[0] != "" {
		e.Mutex.Lock()
		defer e.Mutex.Unlock()

		tmpl := e.Templates[name]
		if tmpl == nil {
			return fmt.Errorf("render: template %s does not exist", name)
		}

		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if err := tmpl.FRender(buf, binding); err != nil {
			return err
		}

		bind := core.AcquireViewContext(binding)
		bind[e.LayoutName] = buf.String()
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}
		return lay.FRender(out, bind)
	}

	// A mustache template is immutable once parsed, so the shared lock only
	// has to cover the lookup.
	e.Mutex.RLock()
	tmpl := e.Templates[name]
	e.Mutex.RUnlock()

	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}
	return tmpl.FRender(out, binding)
}
