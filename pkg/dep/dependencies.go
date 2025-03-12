package dep

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/tidwall/jsonc"
	"gopkg.in/yaml.v3"
)

const (
	ManagerTypeElixir     = "elixir"
	ManagerTypeGo         = "go"
	ManagerTypeJava       = "java"
	ManagerTypeJavaScript = "js"
	ManagerTypePHP        = "php"
	ManagerTypePython     = "python"
	ManagerTypeRuby       = "ruby"
	ManagerTypeRust       = "rust"
)

var AllManagerTypes = []string{
	ManagerTypeElixir,
	ManagerTypeGo,
	ManagerTypeJava,
	ManagerTypeJavaScript,
	ManagerTypePHP,
	ManagerTypePython,
	ManagerTypeRuby,
	ManagerTypeRust,
}

type Dependency struct {
	Vendor     string // The vendor, if any.
	Name       string // The standard package name, which may include the vendor name.
	Constraint string // The version constraint.
	Version    string // The resolved version (e.g. from a lock file).
}

type Manager interface {
	// Init collects data for the manager: parsing files, etc.
	// Implementations may be run multiple times: they should ensure they only read files once.
	Init() error

	// Get finds a specific dependency by name.
	Get(name string) (Dependency, bool)

	// Find looks for dependencies using a wildcard pattern.
	Find(pattern string) []Dependency
}

var managerFuncs = map[string]func(fs.FS, string) Manager{
	ManagerTypeGo:         newGoManager,
	ManagerTypeJava:       newJavaManager,
	ManagerTypeJavaScript: newJSManager,
	ManagerTypePHP:        newPHPManager,
	ManagerTypePython:     newPythonManager,
	ManagerTypeRuby:       newRubyManager,
	ManagerTypeRust:       newRustManager,
	ManagerTypeElixir:     newElixirManager,
}

// GetManager returns a dependency manager for the given type, filesystem and path.
// The caller must then use Manager.Init to ensure files are parsed, when necessary.
func GetManager(managerType string, fsys fs.FS, path string) (Manager, error) {
	if managerFunc, ok := managerFuncs[managerType]; ok {
		return managerFunc(fsys, path), nil
	}
	return nil, fmt.Errorf("manager type not supported: %s", managerType)
}

func parseJSON(fsys fs.FS, path, filename string, dest any) error {
	f, err := fsys.Open(filepath.Join(path, filename))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(dest); err != nil {
		return fmt.Errorf("failed to parse %s as JSON: %w", filepath.Join(path, filename), err)
	}
	return nil
}

func parseJSONC(fsys fs.FS, path, filename string, dest any) error {
	b, err := fs.ReadFile(fsys, filepath.Join(path, filename))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonc.ToJSONInPlace(b), dest); err != nil {
		return fmt.Errorf("failed to parse %s as JSONC: %w", filepath.Join(path, filename), err)
	}
	return nil
}

func parseYAML(fsys fs.FS, path, filename string, dest any) error {
	f, err := fsys.Open(filepath.Join(path, filename))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := yaml.NewDecoder(f).Decode(dest); err != nil {
		return fmt.Errorf("failed to parse %s as YAML: %w", filepath.Join(path, filename), err)
	}
	return nil
}
