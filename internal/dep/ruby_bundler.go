package dep

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

type rubyManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	required map[string]string
	resolved map[string]string
}

func newRubyManager(fsys fs.FS, path string) Manager {
	return &rubyManager{
		fsys: fsys,
		path: path,
	}
}

func (m *rubyManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.parse()
	})
	return err
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

func (m *rubyManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, constraint := range m.required {
		if wildcard.Match(pattern, name) {
			v, _ := m.resolved[name]
			deps = append(deps, Dependency{
				Name:       name,
				Constraint: constraint,
				Version:    v,
			})
		}
	}
	return deps
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
