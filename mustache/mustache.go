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
	fileSystem http.FileSystem
	extension  string
	baseDir    string
	verbose    bool
}

func (p fileSystemPartialProvider) Get(partial string) (string, error) {
	candidates := p.lookupCandidates(partial)
	var firstErr error
	for _, candidate := range candidates {
		buf, err := core.ReadFile(candidate, p.fileSystem)
		if err == nil {
			return string(buf), nil
		}

		if firstErr == nil {
			firstErr = err
		}
		if p.verbose {
			log.Printf("views: partial lookup failed: partial=%q candidate=%q err=%v", partial, candidate, err)
		}
	}

	if p.verbose {
		log.Printf("views: partial not found: partial=%q candidates=%v", partial, candidates)
	}

	if firstErr == nil {
		firstErr = fmt.Errorf("no partial candidates generated")
	}
	return "", fmt.Errorf("render: partial %q does not exist (tried: %s): %w", partial, strings.Join(candidates, ", "), firstErr)
}

func (p fileSystemPartialProvider) lookupCandidates(partial string) []string {
	addCandidate := func(candidates []string, candidate string) []string {
		for _, existing := range candidates {
			if existing == candidate {
				return candidates
			}
		}
		return append(candidates, candidate)
	}

	addExtension := func(raw string) string {
		if strings.HasSuffix(raw, p.extension) {
			return raw
		}
		return raw + p.extension
	}

	base := filepath.ToSlash(strings.TrimSpace(p.baseDir))
	base = strings.TrimSuffix(base, "/")
	if base == "." || base == "/" {
		base = ""
	}

	clean := filepath.ToSlash(strings.TrimSpace(partial))
	clean = strings.TrimPrefix(clean, "./")

	candidates := make([]string, 0, 2)
	if clean != "" {
		candidates = addCandidate(candidates, addExtension(clean))
		if base != "" {
			candidates = addCandidate(candidates, addExtension(path.Join(base, clean)))
		}
	}

	return candidates
}

// New returns a Mustache render engine for Fiber
func New(directory, extension string) *Engine {
	engine := &Engine{
		partialsProvider: &fileSystemPartialProvider{
			extension: extension,
			baseDir:   directory,
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
			baseDir:    "/",
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
	if e.partialsProvider != nil {
		e.partialsProvider.verbose = e.Verbose
	}

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
