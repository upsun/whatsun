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

type elixirManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	required map[string]string
	resolved map[string]string
}

func newElixirManager(fsys fs.FS, path string) Manager {
	return &elixirManager{
		fsys: fsys,
		path: path,
	}
}

func (m *elixirManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.parse()
	})
	return err
}

func (m *elixirManager) parse() error {
	mixfile, err := m.fsys.Open(filepath.Join(m.path, "mix.exs"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer mixfile.Close()
	m.required = parseMixfile(mixfile)

	mixfileLock, err := m.fsys.Open(filepath.Join(m.path, "mix.lock"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer mixfileLock.Close()
	m.resolved = parseMixfileLock(mixfileLock)
	return nil
}

func (m *elixirManager) Get(name string) (Dependency, bool) {
	req, ok := m.required[name]
	if !ok {
		return Dependency{}, false
	}
	return Dependency{
		Name:       name,
		Constraint: req,
		Version:    m.resolved[name],
		IsDirect:   true, // Dependencies from mix.exs are direct
		ToolName:   "mix",
	}, true
}

func (m *elixirManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, constraint := range m.required {
		if wildcard.Match(pattern, name) {
			deps = append(deps, Dependency{
				Name:       name,
				Constraint: constraint,
				Version:    m.resolved[name],
				IsDirect:   true, // Dependencies from mix.exs are direct
				ToolName:   "mix",
			})
		}
	}
	return deps
}

var mixPatt = regexp.MustCompile(`^\s*\{\s*:(\w+),\s*"([^"]+)"`)

func parseMixfile(r io.Reader) map[string]string {
	deps := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "{:") {
			matches := mixPatt.FindStringSubmatch(line)
			if len(matches) > 1 {
				name := matches[1]
				version := ""
				if len(matches) >= 2 {
					version = matches[2]
				}
				deps[name] = version
			}
		}
	}

	return deps
}

var mixLockPatt = regexp.MustCompile(`^\s{2}"([a-zA-Z0-9_-]+)": {:(\w+), :(\w+), "([^"]+)", .*}$`)

func parseMixfileLock(r io.Reader) map[string]string {
	deps := make(map[string]string)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		matches := mixLockPatt.FindStringSubmatch(line)
		if len(matches) >= 4 {
			deps[matches[1]] = matches[4]
		}
	}

	return deps
}
