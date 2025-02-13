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
	constraints, err := parseCargoTOML(manifestFile)
	if err != nil {
		return err
	}

	m.deps = make(map[string]Dependency)
	for name, constraint := range constraints {
		m.deps[name] = Dependency{Name: name, Constraint: constraint}
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
			d.Version = version
			m.deps[name] = d
		} else {
			m.deps[name] = Dependency{Name: name, Version: version}
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
	Dependencies map[string]toml.Primitive `toml:"dependencies"`
}

type cargoLock struct {
	Packages []struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
}

// parseCargoTOML parses the Cargo.toml manifest (dependency names with version constraints).
func parseCargoTOML(r io.Reader) (map[string]string, error) {
	var ct cargoTOML
	md, err := toml.NewDecoder(r).Decode(&ct)
	if err != nil {
		return nil, err
	}

	dependencies := make(map[string]string)
	for name, spec := range ct.Dependencies {
		var asStr string
		if err := md.PrimitiveDecode(spec, &asStr); err == nil {
			dependencies[name] = asStr
			continue
		}
		var asMap map[string]string
		if err := md.PrimitiveDecode(spec, &asMap); err == nil {
			b, _ := json.Marshal(asMap)
			dependencies[name] = string(b)
		}
	}

	return dependencies, nil
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
