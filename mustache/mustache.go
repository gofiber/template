package mustache

import (
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cbroglie/mustache"
	core "github.com/gofiber/template/v2"
	"github.com/gofiber/utils/v2"
	"github.com/valyala/bytebufferpool"
)

// Engine struct
type Engine struct {
	core.Engine
	// partialsFS serves partials that do not live beside the templates
	partialsFS http.FileSystem
	//  templates
	Templates map[string]*mustache.Template
}

// source is one loaded file and every name an include may reach it by.
type source struct {
	name string
	keys []string
	body string
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
	return NewFileSystemPartials(fs, extension, nil)
}

// NewFileSystemPartials returns a Handlebar render engine for Fiber that supports embedded files
func NewFileSystemPartials(fs http.FileSystem, extension string, partialsFS http.FileSystem) *Engine {
	engine := &Engine{
		partialsFS: partialsFS,
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

	// Every template doubles as a partial, so read the whole tree before parsing.
	var partials []source
	if e.partialsFS != nil {
		var err error
		if partials, err = e.collect(e.partialsFS, "/"); err != nil {
			return err
		}
	}

	templates, err := e.collect(e.FileSystem, e.Directory)
	if err != nil {
		return err
	}

	// A partial reaches only what was loaded, so includes stay inside the root.
	bodies := make(map[string]string, len(partials)+len(templates))
	canonical := make(map[string]string, len(partials)+len(templates))
	all := slices.Concat(partials, templates)
	for _, src := range all {
		for _, key := range src.keys {
			bodies[key] = src.body
			canonical[key] = src.name
		}
	}
	// A template always wins its own name, whatever alias another one claims.
	for _, src := range all {
		bodies[src.name] = src.body
		canonical[src.name] = src.name
	}

	provider := &mustache.StaticProvider{Partials: bodies}
	parsed := make(map[string]*mustache.Template, len(all))
	includes := make(map[string][]string, len(all))
	for _, src := range all {
		tmpl, err := mustache.ParseStringPartials(src.body, provider)
		if err != nil {
			return fmt.Errorf("views: template %s: %w", src.name, err)
		}

		parsed[src.name] = tmpl
		includes[src.name] = partialNames(tmpl.Tags())
	}

	if err := checkIncludes(canonical, includes); err != nil {
		return err
	}

	for _, src := range templates {
		e.Templates[src.name] = parsed[src.name]
		if e.Verbose {
			log.Printf("views: parsed template: %s\n", src.name)
		}
	}

	// A load that failed leaves Loaded unset, so the next render retries.
	e.Loaded = true
	return nil
}

// collect reads every template below directory, keeping its source for includes.
func (e *Engine) collect(fs http.FileSystem, directory string) ([]source, error) {
	var sources []source

	walkFn := func(file string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}

		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}

		// Skip file if it does not equal the given template extension
		if !core.HasExtension(file, e.Extension) {
			return nil
		}

		// ./views/html/index.tmpl -> index
		name, err := core.TemplateName(directory, file, e.Extension)
		if err != nil {
			return err
		}

		// Read the file
		// #gosec G304
		buf, err := core.ReadFile(file, fs)
		if err != nil {
			return err
		}

		sources = append(sources, source{
			name: name,
			keys: includeKeys(directory, file, name, e.Extension),
			body: utils.UnsafeString(buf),
		})
		return nil
	}

	if fs != nil {
		if err := core.Walk(fs, directory, walkFn); err != nil {
			return nil, err
		}
		return sources, nil
	}
	if err := filepath.Walk(directory, walkFn); err != nil {
		return nil, err
	}
	return sources, nil
}

// includeKeys lists the names an include may reach a template by.
func includeKeys(directory, file, name, extension string) []string {
	keys := []string{name, "/" + name, strings.TrimSuffix(filepath.ToSlash(file), extension)}

	// Base keeps the qualified form working for an absolute directory too.
	if base := filepath.Base(filepath.Clean(directory)); base != "." && base != "/" && base != string(filepath.Separator) {
		keys = append(keys, base+"/"+name)
	}
	return keys
}

// partialNames collects the partials a template includes, sections included.
func partialNames(tags []mustache.Tag) []string {
	var names []string
	for _, tag := range tags {
		switch tag.Type() {
		case mustache.Partial:
			names = append(names, tag.Name())
		case mustache.Section, mustache.InvertedSection:
			names = append(names, partialNames(tag.Tags())...)
		default:
			// A variable holds no partials, and panics if asked for tags.
		}
	}
	return names
}

// checkIncludes rejects an unknown include, and a cycle that would blow the stack.
func checkIncludes(canonical map[string]string, includes map[string][]string) error {
	const (
		visiting = iota + 1
		visited
	)

	state := make(map[string]int, len(includes))

	var visit func(name string, stack []string) error
	visit = func(name string, stack []string) error {
		if state[name] == visiting {
			return fmt.Errorf("views: partial cycle: %s", strings.Join(append(stack, name), " -> "))
		}
		if state[name] == visited {
			return nil
		}

		state[name] = visiting
		stack = append(stack, name)
		for _, include := range includes[name] {
			target, ok := canonical[include]
			if !ok {
				return fmt.Errorf("views: template %s includes partial %q, which does not exist", name, include)
			}
			if err := visit(target, stack); err != nil {
				return err
			}
		}

		state[name] = visited
		return nil
	}

	for _, name := range slices.Sorted(maps.Keys(includes)) {
		if err := visit(name, nil); err != nil {
			return err
		}
	}
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
