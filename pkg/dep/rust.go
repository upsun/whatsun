package dep

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/IGLOU-EU/go-wildcard/v2"
)

type rustManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	deps     map[string]Dependency
}

func newRustManager(fsys fs.FS, path string) Manager {
	return &rustManager{
		fsys: fsys,
		path: path,
	}
}

func (m *rustManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.parse()
	})
	return err
}

func (m *rustManager) parse() error {
	manifestFile, err := m.fsys.Open(filepath.Join(m.path, "Cargo.toml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
	}
	defer manifestFile.Close()
	regularDeps, devDeps, err := parseCargoTOML(manifestFile)
	if err != nil {
		return err
	}

	m.deps = make(map[string]Dependency)
	// Direct regular dependencies from Cargo.toml
	for name, constraint := range regularDeps {
		m.deps[name] = Dependency{
			Name:       name,
			Constraint: constraint,
			IsDirect:   true,
			ToolName:   "cargo",
		}
	}
	// Direct dev dependencies from Cargo.toml
	for name, constraint := range devDeps {
		m.deps[name] = Dependency{
			Name:       name,
			Constraint: constraint,
			IsDirect:   true,
			IsDevOnly:  true,
			ToolName:   "cargo",
		}
	}

	lockFile, err := m.fsys.Open(filepath.Join(m.path, "Cargo.lock"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer lockFile.Close()
	versions, err := parseCargoLock(lockFile)
	if err != nil {
		return err
	}
	for name, version := range versions {
		if d, ok := m.deps[name]; ok {
			// Update existing dependency (preserve IsDirect status)
			d.Version = version
			m.deps[name] = d
		} else {
			// New dependency only in lock file (indirect)
			m.deps[name] = Dependency{
				Name:     name,
				Version:  version,
				IsDirect: false,
				ToolName: "cargo",
			}
		}
	}

	return nil
}

func (m *rustManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, dep := range m.deps {
		if wildcard.Match(pattern, name) {
			deps = append(deps, dep)
		}
	}
	return deps
}

func (m *rustManager) Get(name string) (Dependency, bool) {
	dep, ok := m.deps[name]
	return dep, ok
}

type cargoTOML struct {
	Dependencies    map[string]toml.Primitive `toml:"dependencies"`
	DevDependencies map[string]toml.Primitive `toml:"dev-dependencies"`
}

type cargoLock struct {
	Packages []struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
}

// parseCargoTOML parses the Cargo.toml manifest (dependency names with version constraints).
func parseCargoTOML(r io.Reader) (regular map[string]string, dev map[string]string, err error) {
	var ct cargoTOML
	md, err := toml.NewDecoder(r).Decode(&ct)
	if err != nil {
		return nil, nil, err
	}

	regular = make(map[string]string)
	dev = make(map[string]string)

	// Parse regular dependencies
	for name, spec := range ct.Dependencies {
		var asStr string
		if err := md.PrimitiveDecode(spec, &asStr); err == nil {
			regular[name] = asStr
			continue
		}
		var asMap map[string]string
		if err := md.PrimitiveDecode(spec, &asMap); err == nil {
			b, _ := json.Marshal(asMap)
			regular[name] = string(b)
		}
	}

	// Parse dev dependencies
	for name, spec := range ct.DevDependencies {
		var asStr string
		if err := md.PrimitiveDecode(spec, &asStr); err == nil {
			dev[name] = asStr
			continue
		}
		var asMap map[string]string
		if err := md.PrimitiveDecode(spec, &asMap); err == nil {
			b, _ := json.Marshal(asMap)
			dev[name] = string(b)
		}
	}

	return regular, dev, nil
}

// parseCargoLock parses the Cargo.lock file (dependency names with resolved versions).
func parseCargoLock(r io.Reader) (map[string]string, error) {
	var cl cargoLock
	_, err := toml.NewDecoder(r).Decode(&cl)
	if err != nil {
		return nil, err
	}

	dependencies := make(map[string]string)
	for _, pkg := range cl.Packages {
		dependencies[pkg.Name] = pkg.Version
	}

	return dependencies, nil
}
