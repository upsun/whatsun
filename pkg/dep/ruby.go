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
	constraint, hasConstraint := m.required[name]
	version, hasVersion := m.resolved[name]
	if !hasConstraint && !hasVersion {
		return Dependency{}, false
	}
	return Dependency{
		Name:       name,
		Constraint: constraint,
		Version:    version,
		IsDirect:   hasConstraint, // Direct if it has a constraint from Gemfile
		ToolName:   "bundler",
	}, true
}

func (m *rubyManager) Find(pattern string) []Dependency {
	seen := make(map[string]struct{})
	var deps []Dependency
	// First, add all with constraints (direct dependencies)
	for name, constraint := range m.required {
		if wildcard.Match(pattern, name) {
			deps = append(deps, Dependency{
				Name:       name,
				Constraint: constraint,
				Version:    m.resolved[name],
				IsDirect:   true,
				ToolName:   "bundler",
			})
			seen[name] = struct{}{}
		}
	}
	// Then, add any resolved-only deps not already included (indirect dependencies)
	for name, version := range m.resolved {
		if _, already := seen[name]; !already && wildcard.Match(pattern, name) {
			deps = append(deps, Dependency{
				Name:     name,
				Version:  version,
				IsDirect: false,
				ToolName: "bundler",
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
