// Package generator renders and writes projects without executing external tools.
package generator

import (
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aoamusat/hexgogen/templates"
)

const templateModule = "github.com/aoamusat/hexgogen"

var (
	namePattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	hostPattern         = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*$`)
	versionPattern      = regexp.MustCompile(`^v[0-9.]+$`)
	gopkgVersionPattern = regexp.MustCompile(`\.v(0|[1-9][0-9]*)(-unstable)?$`)
)

// Plan describes the rendered files and destination without changing the filesystem.
// Generate rechecks the destination before writing to protect against conflicts
// introduced after the preview was created.
type Plan struct {
	destination     string
	module          string
	files           []renderedFile
	createDirectory bool
}

func (p *Plan) Destination() string { return p.destination }
func (p *Plan) Module() string      { return p.module }

// Files returns the generated paths relative to Destination, in deterministic order.
func (p *Plan) Files() []string {
	names := make([]string, len(p.files))
	for i, file := range p.files {
		names[i] = file.name
	}
	return names
}

// PlanNew validates a new project and renders its files without creating it.
func PlanNew(parent, name, module string) (*Plan, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	destination, err := filepath.Abs(filepath.Join(parent, name))
	if err != nil {
		return nil, err
	}
	files, err := render(name, module)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(filepath.Dir(destination))
	if err != nil {
		return nil, fmt.Errorf("inspect project parent: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("project parent must be a directory")
	}
	if _, err := os.Lstat(destination); err == nil {
		return nil, fmt.Errorf("project destination already exists: %s", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return &Plan{destination: destination, module: module, files: files, createDirectory: true}, nil
}

// PlanInit validates an existing destination, preserving optional Git metadata.
func PlanInit(directory, module string) (*Plan, error) {
	destination, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(destination)
	if err := validateName(name); err != nil {
		return nil, err
	}
	files, err := render(name, module)
	if err != nil {
		return nil, err
	}
	root, err := openDestination(destination)
	if err != nil {
		return nil, err
	}
	if err := root.Close(); err != nil {
		return nil, err
	}
	return &Plan{destination: destination, module: module, files: files}, nil
}

// Generate applies the plan. Existing files are never overwritten.
func (p *Plan) Generate() (string, error) {
	if p.createDirectory {
		if err := os.Mkdir(p.destination, 0755); err != nil {
			return "", fmt.Errorf("create project directory: %w", err)
		}
	}
	if err := populate(p.destination, p.files); err != nil {
		if p.createDirectory {
			// Remove only an empty directory; never recursively delete user files.
			err = errors.Join(err, os.Remove(p.destination))
		}
		return "", err
	}
	return p.destination, nil
}

// New creates a project inside parent. An existing target is always refused.
func New(parent, name, module string) (string, error) {
	plan, err := PlanNew(parent, name, module)
	if err != nil {
		return "", err
	}
	return plan.Generate()
}

// Init initializes an existing directory containing only optional Git metadata.
func Init(directory, module string) (string, error) {
	plan, err := PlanInit(directory, module)
	if err != nil {
		return "", err
	}
	return plan.Generate()
}

type renderedFile struct {
	name string
	data []byte
}

func render(name, module string) ([]renderedFile, error) {
	if err := validateModule(module); err != nil {
		return nil, err
	}
	var files []renderedFile
	err := fs.WalkDir(templates.Files, ".", func(source string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := templates.Files.ReadFile(source)
		if err != nil {
			return err
		}
		relative := strings.TrimSuffix(source, ".tmpl")
		text := strings.NewReplacer(templateModule, module, "{{PROJECT_NAME}}", name).Replace(string(data))
		data = []byte(text)
		if strings.HasSuffix(relative, ".go") {
			data, err = format.Source(data)
			if err != nil {
				return fmt.Errorf("format %s: %w", relative, err)
			}
		}
		files = append(files, renderedFile{name: relative, data: data})
		return nil
	})
	return files, err
}

func openDestination(directory string) (*os.Root, error) {
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, fmt.Errorf("open destination: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("destination must be a real directory, not a symlink")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" && (entry.IsDir() || entry.Type().IsRegular()) {
			continue
		}
		_ = root.Close()
		return nil, errors.New("destination contains files other than Git metadata; no files were changed")
	}
	return root, nil
}

// populate confines writes to an opened directory and uses exclusive creation.
// On failure it removes only paths this invocation created, in reverse order.
func populate(directory string, files []renderedFile) (result error) {
	root, err := openDestination(directory)
	if err != nil {
		return err
	}
	defer root.Close()
	var created []string
	defer func() {
		if result == nil {
			return
		}
		for i := len(created) - 1; i >= 0; i-- {
			if err := root.Remove(created[i]); err != nil {
				result = errors.Join(result, fmt.Errorf("clean up %s: %w", created[i], err))
			}
		}
	}()
	for _, file := range files {
		parent := path.Dir(file.name)
		if parent != "." {
			parts := strings.Split(parent, "/")
			for i := range parts {
				dir := strings.Join(parts[:i+1], "/")
				if err := root.Mkdir(dir, 0755); err == nil {
					created = append(created, dir)
				} else if !errors.Is(err, os.ErrExist) {
					return err
				}
			}
		}
		output, err := root.OpenFile(file.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return fmt.Errorf("create %s: %w", file.name, err)
		}
		created = append(created, file.name)
		_, writeErr := output.Write(file.data)
		closeErr := output.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return fmt.Errorf("write %s: %w", file.name, err)
		}
	}
	return nil
}

func validateName(name string) error {
	if !namePattern.MatchString(name) || strings.HasSuffix(name, ".") || reservedName(name) {
		return errors.New("project name must start with a letter or digit and contain only letters, digits, dots, hyphens, or underscores; reserved device names and trailing dots are not allowed")
	}
	return nil
}

func validateModule(module string) error {
	if module == "" || strings.Contains(module, "\\") {
		return errors.New("module path is required and must use forward slashes")
	}
	for _, part := range strings.Split(module, "/") {
		if !namePattern.MatchString(part) || strings.HasSuffix(part, ".") || strings.Contains(part, "..") || reservedName(part) {
			return fmt.Errorf("invalid module path %q", module)
		}
	}
	first := strings.SplitN(module, "/", 2)[0]
	if strings.Contains(first, ".") && !hostPattern.MatchString(first) {
		return errors.New("module hostname must use lowercase letters, digits, dots, or hyphens")
	}
	last := path.Base(module)
	if strings.Contains(module, "/") && versionPattern.MatchString(last) {
		if strings.Contains(last, ".") || strings.HasPrefix(last, "v0") || last == "v1" {
			return errors.New("module version suffix must be /vN for N >= 2, without leading zeros")
		}
	}
	if strings.HasPrefix(module, "gopkg.in/") && !gopkgVersionPattern.MatchString(module) {
		return errors.New("gopkg.in module paths require a .vN suffix")
	}
	if module == "std" || module == "cmd" {
		return errors.New("module path is reserved by Go")
	}
	return nil
}

func reservedName(name string) bool {
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}
