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
	"github.com/gofiber/utils/v2"
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
	return utils.UnsafeString(buf), err
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
		source := utils.UnsafeString(buf)
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

	var err error
	if e.FileSystem != nil {
		err = core.Walk(e.FileSystem, e.Directory, walkFn)
	} else {
		err = filepath.Walk(e.Directory, walkFn)
	}
	if err != nil {
		return err
	}

	// A load that failed leaves Loaded unset, so the next render retries.
	e.Loaded = true
	return nil
}

// Render will render the template by name
func (e *Engine) Render(out io.Writer, name string, binding interface{}, layout ...string) error {
	// Check if templates need to be loaded/reloaded
	if e.PreRenderCheck() {
		if err := e.Load(); err != nil {
			return err
		}
	}

	// A mustache template is immutable once parsed, so renders share the lock.
	e.Mutex.RLock()
	defer e.Mutex.RUnlock()

	tmpl := e.Templates[name]
	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	if len(layout) > 0 && layout[0] != "" {
		lay := e.Templates[layout[0]]
		if lay == nil {
			return fmt.Errorf("render: layout %s does not exist", layout[0])
		}

		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)
		if err := tmpl.FRender(buf, binding); err != nil {
			return err
		}

		// Our own context: the embed key must not land in the caller's map.
		bind := core.NewViewContext(binding, 1)
		bind[e.LayoutName] = buf.String()
		return lay.FRender(out, bind)
	}
	return tmpl.FRender(out, binding)
}
