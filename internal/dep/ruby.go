package dep

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

type rubyManager struct {
	fsys fs.FS
	path string

	required map[string]string
	resolved map[string]string
}

func newRubyManager(fsys fs.FS, path string) (Manager, error) {
	m := &rubyManager{
		fsys: fsys,
		path: path,
	}
	if err := m.parse(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *rubyManager) parse() error {
	gemfile, err := m.fsys.Open(filepath.Join(m.path, "Gemfile"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer gemfile.Close()
	m.required = parseGemfile(gemfile)

	gemfileLock, err := m.fsys.Open(filepath.Join(m.path, "Gemfile.lock"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer gemfileLock.Close()
	m.resolved = parseGemfileLock(gemfileLock)
	return nil
}

func (m *rubyManager) Get(name string) (Dependency, bool) {
	req, ok := m.required[name]
	if !ok {
		return Dependency{}, false
	}
	v, _ := m.resolved[name]
	return Dependency{
		Name:       name,
		Constraint: req,
		Version:    v,
	}, true
}

func (m *rubyManager) Find(pattern string) ([]Dependency, error) {
	matches, err := matchDependencyKey(m.required, pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	var deps = make([]Dependency, 0, len(matches))
	for _, match := range matches {
		v, _ := m.resolved[match]
		deps = append(deps, Dependency{
			Name:       match,
			Constraint: m.required[match],
			Version:    v,
		})
	}
	return deps, nil
}

var gemPatt = regexp.MustCompile(`^gem\s+['"]([a-zA-Z0-9_-]+)['"](,\s*['"]([^'"]+)['"])?`)

func parseGemfile(r io.Reader) map[string]string {
	deps := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "gem ") {
			matches := gemPatt.FindStringSubmatch(line)
			if len(matches) > 1 {
				name := matches[1]
				version := ""
				if len(matches) > 3 {
					version = matches[3]
				}
				deps[name] = version
			}
		}
	}

	return deps
}

var bundlerDepPatt = regexp.MustCompile(`^\s{4}([a-zA-Z0-9_-]+) \(([^)]+)\)$`)

func parseGemfileLock(r io.Reader) map[string]string {
	deps := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		matches := bundlerDepPatt.FindStringSubmatch(line)
		if len(matches) > 2 {
			deps[matches[1]] = matches[2]
		}
	}

	return deps
}
