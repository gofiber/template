package django

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/flosch/pongo2/v6"
	core "github.com/gofiber/template/v2"
)

var (
	pongo2AutoescapeMu    sync.Mutex
	pongo2AutoescapeState = true
)

// Engine struct
type Engine struct {
	core.Engine
	// forward the base path to the template Engine
	forwardPath bool
	// set auto escape globally
	autoEscape bool
	// templates
	Templates map[string]*pongo2.Template
	// sandbox restrictions
	bannedTags    []string
	bannedFilters []string
}

// This helper function is used to avoid duplication in public constructors.
func (e *Engine) initialize(directory, extension string, fs http.FileSystem) {
	e.Left = "{{"
	e.Right = "}}"
	e.Directory = directory
	e.Extension = extension
	e.LayoutName = "embed"
	e.Funcmap = make(map[string]interface{})
	e.autoEscape = true
	e.FileSystem = fs
}

// New creates a new Engine with a directory and extension.
func New(directory, extension string) *Engine {
	engine := &Engine{}
	engine.initialize(directory, extension, nil)
	return engine
}

// NewFileSystem creates a new Engine with a file system and extension.
func NewFileSystem(fs http.FileSystem, extension string) *Engine {
	engine := &Engine{}
	engine.initialize("/", extension, fs)
	return engine
}

// NewPathForwardingFileSystem creates a new Engine with path forwarding,
// using a file system, directory, and extension.
func NewPathForwardingFileSystem(fs http.FileSystem, directory, extension string) *Engine {
	engine := &Engine{forwardPath: true}
	engine.initialize(directory, extension, fs)
	return engine
}

// Load parses the templates to the engine.
func (e *Engine) Load() error {
	// race safe
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	e.Templates = make(map[string]*pongo2.Template)
	pongoloader, err := e.newTemplateLoader()
	if err != nil {
		return err
	}

	pongoset := pongo2.NewSet("fiber-django", pongoloader)
	// Set template settings
	pongoset.Globals.Update(e.Funcmap)

	if err := e.applySandbox(pongoset); err != nil {
		return err
	}

	// Loop trough each Directory and register template files
	walkFn := func(path string, info os.FileInfo, err error) error {
		// Return error if exist
		if err != nil {
			return err
		}

		// Skip file if it's a directory or has no file info
		if info == nil || info.IsDir() {
			return nil
		}

		// Skip file if it does not equal the given template Extension
		if len(e.Extension) >= len(path) || path[len(path)-len(e.Extension):] != e.Extension {
			return nil
		}

		filename, name, err := templateNames(e.Directory, path, e.Extension)
		if err != nil {
			return err
		}

		return withScopedAutoescape(e.autoEscape, func() error {
			tmpl, err := pongoset.FromFile(filename)
			if err != nil {
				return err
			}
			e.Templates[name] = tmpl

			if e.Verbose {
				log.Printf("views: parsed template: %s\n", name)
			}
			return nil
		})
	}

	// notify Engine that we parsed all templates
	e.Loaded = true

	if e.FileSystem != nil {
		return core.Walk(e.FileSystem, e.Directory, walkFn)
	}
	return filepath.Walk(e.Directory, walkFn)
}

// getPongoBinding creates a pongo2.Context containing
// only valid identifiers from a binding interface.
//
// It supports the following types:
// - pongo2.Context
// - map[string]interface{}
// It returns nil if the binding is not one of the supported types.
func getPongoBinding(binding interface{}) pongo2.Context {
	if binding == nil {
		return nil
	}

	switch binds := binding.(type) {
	case pongo2.Context:
		return sanitizePongoContext(binds)
	case map[string]interface{}:
		return sanitizePongoContext(binds)
	}

	value := reflect.ValueOf(binding)
	if value.Kind() != reflect.Map || value.IsNil() {
		return nil
	}

	if value.Type().Key().Kind() != reflect.String {
		return nil
	}

	bind := make(pongo2.Context, value.Len())
	for _, key := range value.MapKeys() {
		strKey := key.String()
		if !isValidKey(strKey) {
			continue
		}
		bind[strKey] = value.MapIndex(key).Interface()
	}

	return bind
}

func sanitizePongoContext(data map[string]interface{}) pongo2.Context {
	if len(data) == 0 {
		return make(pongo2.Context)
	}

	bind := make(pongo2.Context, len(data))
	for key, value := range data {
		if !isValidKey(key) {
			continue
		}
		bind[key] = value
	}
	return bind
}

// isValidKey checks if the key is valid
//
// Valid keys match the following regex: [a-zA-Z0-9_]+
func isValidKey(key string) bool {
	for _, ch := range key {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '_' {
			return false
		}
	}
	return true
}

// SetAutoEscape sets the auto-escape property of the template engine
func (e *Engine) SetAutoEscape(autoEscape bool) {
	e.autoEscape = autoEscape
}

// BanTag applies a pongo2 tag restriction before templates are loaded.
func (e *Engine) BanTag(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("tag name is required")
	}

	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	if !containsString(e.bannedTags, name) {
		e.bannedTags = append(e.bannedTags, name)
	}
	e.Loaded = false
	return nil
}

// BanFilter applies a pongo2 filter restriction before templates are loaded.
func (e *Engine) BanFilter(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("filter name is required")
	}

	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	if !containsString(e.bannedFilters, name) {
		e.bannedFilters = append(e.bannedFilters, name)
	}
	e.Loaded = false
	return nil
}

// Sandbox applies the pongo2-recommended restrictions for user-generated templates.
func (e *Engine) Sandbox() error {
	for _, tag := range []string{"include", "import", "ssi", "extends"} {
		if err := e.BanTag(tag); err != nil {
			return err
		}
	}
	return e.BanFilter("safe")
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
	tmpl, ok := e.Templates[name]
	e.Mutex.RUnlock()

	if !ok {
		return fmt.Errorf("template %s does not exist", name)
	}

	// Lock while executing layout
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	var renderErr error
	err := withScopedAutoescape(e.autoEscape, func() error {
		bind := getPongoBinding(binding)
		parsed, err := tmpl.Execute(bind)
		if err != nil {
			return err
		}

		if len(layout) > 0 && layout[0] != "" {
			if bind == nil {
				bind = make(map[string]interface{}, 1)
			}

			// Workaround for custom {{embed}} tag
			// Mark the `embed` variable as safe
			// it has already been escaped above
			// e.LayoutName will be 'embed'
			safeEmbed := pongo2.AsSafeValue(parsed)

			// Add the safe value to the binding map
			bind[e.LayoutName] = safeEmbed

			lay := e.Templates[layout[0]]
			if lay == nil {
				return fmt.Errorf("LayoutName %s does not exist", layout[0])
			}
			return lay.ExecuteWriter(bind, out)
		}
		_, renderErr = out.Write([]byte(parsed))
		return renderErr
	})
	if err != nil {
		return err
	}
	return nil
}

func (e *Engine) newTemplateLoader() (pongo2.TemplateLoader, error) {
	if e.FileSystem != nil {
		baseDir := ""
		if e.forwardPath {
			baseDir = e.Directory
		}
		return newSecureHTTPTemplateLoader(e.FileSystem, baseDir, e.Extension)
	}
	return newSecureLocalTemplateLoader(e.Directory, e.Extension)
}

func (e *Engine) applySandbox(set *pongo2.TemplateSet) error {
	for _, name := range e.bannedTags {
		if err := set.BanTag(name); err != nil {
			return err
		}
	}
	for _, name := range e.bannedFilters {
		if err := set.BanFilter(name); err != nil {
			return err
		}
	}
	return nil
}

func withScopedAutoescape(autoEscape bool, fn func() error) error {
	pongo2AutoescapeMu.Lock()
	prev := pongo2AutoescapeState
	pongo2AutoescapeState = autoEscape
	pongo2.SetAutoescape(autoEscape)
	defer func() {
		pongo2.SetAutoescape(prev)
		pongo2AutoescapeState = prev
		pongo2AutoescapeMu.Unlock()
	}()
	return fn()
}

func templateNames(baseDir, filePath, extension string) (filename, name string, err error) {
	rel, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		return "", "", err
	}
	filename = filepath.ToSlash(rel)
	name = strings.TrimSuffix(filename, extension)
	return filename, name, nil
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

type secureLocalTemplateLoader struct {
	root string
	ext  string
}

func newSecureLocalTemplateLoader(root, ext string) (*secureLocalTemplateLoader, error) {
	if !filepath.IsAbs(root) {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		root = absRoot
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("the given path %q is not a directory", root)
	}

	return &secureLocalTemplateLoader{root: filepath.Clean(root), ext: ext}, nil
}

func (l *secureLocalTemplateLoader) Abs(base, name string) string {
	return l.resolve(base, name)
}

func (l *secureLocalTemplateLoader) Get(name string) (io.Reader, error) {
	if name == "" {
		return nil, fmt.Errorf("template path must stay within %q and end with %q", l.root, l.ext)
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (l *secureLocalTemplateLoader) resolve(base, name string) string {
	if filepath.IsAbs(name) {
		resolved := filepath.Clean(name)
		if !isWithinRoot(l.root, resolved) || !hasAllowedExtension(filepath.Ext(resolved), l.ext) {
			return ""
		}
		return resolved
	}

	rootDir := l.root
	if base != "" {
		basePath := filepath.Clean(filepath.FromSlash(base))
		if filepath.IsAbs(basePath) {
			if !isWithinRoot(l.root, basePath) {
				return ""
			}
			rootDir = filepath.Dir(basePath)
		} else {
			rootDir = filepath.Join(l.root, filepath.Dir(basePath))
		}
	}

	resolved := filepath.Clean(filepath.Join(rootDir, filepath.FromSlash(name)))
	if !isWithinRoot(l.root, resolved) || !hasAllowedExtension(filepath.Ext(resolved), l.ext) {
		return ""
	}
	return resolved
}

type secureHTTPTemplateLoader struct {
	fs      http.FileSystem
	baseDir string
	ext     string
}

func newSecureHTTPTemplateLoader(fs http.FileSystem, baseDir, ext string) (*secureHTTPTemplateLoader, error) {
	if fs == nil {
		return nil, errors.New("http filesystem cannot be nil")
	}
	return &secureHTTPTemplateLoader{
		fs:      fs,
		baseDir: normalizeHTTPBase(baseDir),
		ext:     ext,
	}, nil
}

func (l *secureHTTPTemplateLoader) Abs(base, name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}

	name = strings.ReplaceAll(name, `\`, "/")
	if path.IsAbs(name) {
		var ok bool
		name, ok = trimHTTPBase(name, l.baseDir)
		if !ok {
			return ""
		}
	}

	baseDir := "."
	if base != "" {
		basePath := strings.ReplaceAll(base, `\`, "/")
		if path.IsAbs(basePath) {
			var ok bool
			basePath, ok = trimHTTPBase(basePath, l.baseDir)
			if !ok {
				return ""
			}
		}
		baseDir = path.Dir(basePath)
	}

	resolved := path.Clean(path.Join(baseDir, name))
	if resolved == "." || resolved == ".." || strings.HasPrefix(resolved, "../") || !hasAllowedExtension(path.Ext(resolved), l.ext) {
		return ""
	}
	return resolved
}

func (l *secureHTTPTemplateLoader) Get(name string) (io.Reader, error) {
	if name == "" {
		return nil, fmt.Errorf("template path must stay within %q and end with %q", l.baseDir, l.ext)
	}

	fullPath := name
	if l.baseDir != "" {
		fullPath = path.Join(l.baseDir, name)
	}
	return l.fs.Open(fullPath)
}

func normalizeHTTPBase(baseDir string) string {
	if strings.TrimSpace(baseDir) == "" {
		return ""
	}
	baseDir = strings.ReplaceAll(baseDir, `\`, "/")
	cleaned := path.Clean(baseDir)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func trimHTTPBase(candidate, baseDir string) (string, bool) {
	candidate = path.Clean(candidate)
	if baseDir == "" {
		return strings.TrimPrefix(candidate, "/"), true
	}
	if candidate == baseDir {
		return "", true
	}
	prefix := baseDir + "/"
	if !strings.HasPrefix(candidate, prefix) {
		return "", false
	}
	return strings.TrimPrefix(candidate, prefix), true
}

func hasAllowedExtension(ext, allowed string) bool {
	return allowed == "" || ext == allowed
}

func isWithinRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
