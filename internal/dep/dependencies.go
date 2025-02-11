package dep

import (
	"encoding/json"
	"fmt"
	"github.com/IGLOU-EU/go-wildcard/v2"
	"io/fs"
	"path/filepath"
)

const (
	ManagerTypePHP        = "php"
	ManagerTypeJavaScript = "js"
	ManagerTypeGo         = "go"
	ManagerTypePython     = "python"
	ManagerTypeRuby       = "ruby"
	ManagerTypeJava       = "java"
)

type Dependency struct {
	Vendor     string // The vendor, if any.
	Name       string // The standard package name, which may include the vendor name.
	Constraint string // The version constraint.
	Version    string // The resolved version (e.g. from a lock file).
}

type Manager interface {
	Get(name string) (Dependency, bool)
	Find(pattern string) ([]Dependency, error)
}

func GetManager(managerType string, fsys fs.FS, path string) (Manager, error) {
	switch managerType {
	case ManagerTypePHP:
		return newPHPManager(fsys, path)
	case ManagerTypeJavaScript:
		return newJSManager(fsys, path)
	case ManagerTypeGo:
		return newGoManager(fsys, path)
	case ManagerTypePython:
		return newPythonManager(fsys, path)
	case ManagerTypeRuby:
		return newRubyManager(fsys, path)
	case ManagerTypeJava:
		return newJavaManager(fsys, path)
	}
	return nil, fmt.Errorf("manager type not (yet) supported: %s", managerType)
}

// matchDependencyKey returns a value if a key is found in a map, or an empty string.
// The key can contain "*" as a wildcard.
// It returns all the matching keys.
func matchDependencyKey[M ~map[string]V, V any](m M, key string) ([]string, error) {
	if m == nil {
		return nil, nil
	}
	if _, ok := m[key]; ok {
		return []string{key}, nil
	}

	var matches = make([]string, 0, len(m))
	for k := range m {
		if wildcard.Match(key, k) {
			matches = append(matches, k)
		}
	}
	return matches, nil
}

func parseJSON(fsys fs.FS, path, filename string, dest any) error {
	f, err := fsys.Open(filepath.Join(path, filename))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(dest); err != nil {
		return fmt.Errorf("failed to parse %s: %w", filepath.Join(path, filename), err)
	}
	return nil
}
