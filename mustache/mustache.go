package mustache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

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
	root       string
	fileSystem http.FileSystem
	extension  string
}

func (p fileSystemPartialProvider) Get(name string) (string, error) {
	cleanName, err := sanitizePartialName(name)
	if err != nil {
		return "", err
	}

	filename := cleanName + p.extension
	if p.fileSystem == nil {
		filename = filepath.Join(p.root, filepath.FromSlash(filename))
	} else if p.root != "" {
		filename = path.Join(p.root, filename)
	}

	buf, err := core.ReadFile(filename, p.fileSystem)
	return string(buf), err
}

func sanitizePartialName(name string) (string, error) {
	cleanName := strings.ReplaceAll(strings.TrimSpace(name), `\`, "/")
	cleanName = path.Clean(cleanName)

	switch {
	case cleanName == "." || cleanName == "..":
		return "", fmt.Errorf("mustache: invalid partial path %q", name)
	case path.IsAbs(cleanName):
		return "", fmt.Errorf("mustache: invalid partial path %q", name)
	case strings.HasPrefix(cleanName, "../"):
		return "", fmt.Errorf("mustache: invalid partial path %q", name)
	}

	return cleanName, nil
}

// New returns a Mustache render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		partialsProvider: &fileSystemPartialProvider{
			root:      ".",
			extension: extension,
		},
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
		// name = strings.Replace(name, e.extension, "", -1)
		// Read the file
		// #gosec G304
		buf, err := core.ReadFile(path, e.FileSystem)
		if err != nil {
			return err
		}

		// Create new template associated with the current one
		// This enable use to invoke other templates {{ template .. }}
		var tmpl *mustache.Template
		if e.partialsProvider != nil {
			tmpl, err = mustache.ParseStringPartials(string(buf), e.partialsProvider)
		} else {
			tmpl, err = mustache.ParseString(string(buf))
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

	// Acquire read lock for accessing the template
	e.Mutex.RLock()
	tmpl := e.Templates[name]
	e.Mutex.RUnlock()

	if tmpl == nil {
		return fmt.Errorf("render: template %s does not exist", name)
	}

	// Lock while executing layout
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	if len(layout) > 0 && layout[0] != "" {
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
	return tmpl.FRender(out, binding)
}
