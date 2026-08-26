package mustache

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	slashpath "path"
	"path/filepath"
	"strings"

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

// fileSystemPartialProvider resolves the partials a template includes, either
// from disk or from the engine's http.FileSystem.
type fileSystemPartialProvider struct {
	fileSystem http.FileSystem
	extension  string
	// baseDir is the engine directory partials are confined to. It is empty
	// when fileSystem is set, which confines them to its own root instead.
	baseDir string
	verbose bool
}

// Get returns the source of a partial, trying each candidate path in turn and
// reporting every path it tried when none of them holds the partial.
func (p fileSystemPartialProvider) Get(partial string) (string, error) {
	candidates := p.lookupCandidates(partial)
	if len(candidates) == 0 {
		if p.verbose {
			log.Printf("views: partial rejected: partial=%q", partial)
		}
		return "", fmt.Errorf("render: partial %q is not a valid path inside the template root", partial)
	}

	var firstErr error
	for _, candidate := range candidates {
		buf, err := core.ReadFile(candidate, p.fileSystem)
		if err == nil {
			return utils.UnsafeString(buf), nil
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
	return "", fmt.Errorf("render: partial %q does not exist (tried: %s): %w", partial, strings.Join(candidates, ", "), firstErr)
}

// lookupCandidates lists the files that may hold the partial. The engine
// directory comes first so a configured directory wins over the process working
// directory, and the name as written follows it to keep templates that already
// spell out the full path working. Every candidate stays inside the engine
// directory.
func (p fileSystemPartialProvider) lookupCandidates(partial string) []string {
	name := sanitizePartial(partial)
	if name == "" {
		return nil
	}
	if !core.HasExtension(name, p.extension) {
		name += p.extension
	}

	// An http.FileSystem addresses its files with slash paths below its own
	// root, and refuses to serve anything above it.
	if p.fileSystem != nil {
		return []string{name}
	}

	// On disk the candidates are operating system paths, so that a Windows UNC
	// root such as \\server\share\views keeps its share.
	local := filepath.FromSlash(name)
	base := localBaseDir(p.baseDir)
	if base == "" {
		// The engine reads from the working directory, which the name is
		// already relative to.
		return []string{local}
	}

	candidates := []string{filepath.Join(base, local)}
	if withinDir(base, local) {
		candidates = append(candidates, local)
	}
	return candidates
}

// sanitizePartial normalizes a partial reference and rejects the ones that would
// leave the template root behind, either through an absolute path or through
// ".." segments.
func sanitizePartial(partial string) string {
	name := filepath.ToSlash(strings.TrimSpace(partial))
	if name == "" {
		return ""
	}

	if slashpath.IsAbs(name) || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return ""
	}

	name = slashpath.Clean(name)
	if name == "." || name == ".." || strings.HasPrefix(name, "../") {
		return ""
	}
	return name
}

// localBaseDir cleans the engine directory into an operating system path
// prefix. The working directory needs no prefix, because the names are already
// relative to it.
func localBaseDir(directory string) string {
	base := strings.TrimSpace(directory)
	if base == "" {
		return ""
	}

	base = filepath.Clean(base)
	if base == "." {
		return ""
	}
	return base
}

// withinDir reports whether a working directory relative name lands inside dir.
// Both are compared lexically, the way http.Dir confines its own lookups.
func withinDir(dir, name string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	absName, err := filepath.Abs(name)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absDir, absName)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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

	e.Loaded = false
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
