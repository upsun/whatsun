package dep

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/IGLOU-EU/go-wildcard/v2"
)

type pythonManager struct {
	fsys fs.FS
	path string

	initOnce     sync.Once
	requirements map[string]string
}

func newPythonManager(fsys fs.FS, path string) Manager {
	return &pythonManager{
		fsys: fsys,
		path: path,
	}
}

func (m *pythonManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.parse()
	})
	return err
}

func (m *pythonManager) parseFile(filename string, parseFunc func(io.Reader) (map[string]string, error)) error {
	f, err := m.fsys.Open(filepath.Join(m.path, filename))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()
	reqs, err := parseFunc(f)
	if err != nil {
		return err
	}
	if m.requirements == nil {
		m.requirements = make(map[string]string)
	}
	for k, v := range reqs {
		m.requirements[k] = v
	}
	return nil
}

func (m *pythonManager) parse() error {
	if err := m.parseFile("requirements.txt", parseRequirementsTXT); err != nil {
		return err
	}
	if err := m.parseFile("Pipfile", parsePipfile); err != nil {
		return err
	}
	if err := m.parseFile("pyproject.toml", parsePyprojectTOML); err != nil {
		return err
	}
	return nil
}

func (m *pythonManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, constraint := range m.requirements {
		if wildcard.Match(pattern, name) {
			deps = append(deps, Dependency{
				Name:       name,
				Constraint: constraint,
			})
		}
	}
	return deps
}

func (m *pythonManager) Get(name string) (Dependency, bool) {
	constraint, ok := m.requirements[name]
	if !ok {
		return Dependency{}, false
	}
	return Dependency{
		Name:       name,
		Constraint: constraint,
	}, true
}

var pipPattern = regexp.MustCompile(`^([\w-]+)([<>=!~]+[\w.-]*)?$`)

// parseRequirementsTXT parses a requirements.txt file
func parseRequirementsTXT(r io.Reader) (map[string]string, error) {
	dependencies := make(map[string]string)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := pipPattern.FindStringSubmatch(line)
		if len(matches) > 1 {
			packageName := matches[1]
			versionConstraint := ""
			if len(matches) > 2 {
				versionConstraint = matches[2]
			}
			dependencies[packageName] = versionConstraint
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dependencies, nil
}

var pipEnvPattern = regexp.MustCompile(`^\s*"?([\w-]+)"?\s*=\s*"([^"]+)"`)

// parsePipfile extracts dependencies from a Pipfile
func parsePipfile(r io.Reader) (map[string]string, error) {
	dependencies := make(map[string]string)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := pipEnvPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			dependencies[matches[1]] = matches[2]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dependencies, nil
}

// parsePyprojectTOML parses pyproject.toml dependencies (Poetry and PEP 621)
func parsePyprojectTOML(r io.Reader) (map[string]string, error) {
	type PyProject struct {
		Project struct {
			Dependencies []string `toml:"dependencies"`
		} `toml:"project"`
		Tool struct {
			Poetry struct {
				Dependencies map[string]any `toml:"dependencies"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}

	var pyProject PyProject
	_, err := toml.NewDecoder(r).Decode(&pyProject)
	if err != nil {
		return nil, err
	}

	dependencies := make(map[string]string)

	// Handle Poetry dependencies
	for pkg, version := range pyProject.Tool.Poetry.Dependencies {
		switch v := version.(type) {
		case string:
			dependencies[pkg] = v
		case map[string]any:
			dependencies[pkg] = v["version"].(string)
		default:
			return nil, fmt.Errorf("unrecognized poetry version type %T: %v", version, version)
		}
	}

	// Handle PEP 508/621 dependencies (pip and Poetry 2.0)
	for _, dep := range pyProject.Project.Dependencies {
		matches := pipPattern.FindStringSubmatch(dep)
		if len(matches) > 1 {
			packageName := matches[1]
			versionConstraint := ""
			if len(matches) > 2 {
				versionConstraint = matches[2]
			}
			dependencies[packageName] = versionConstraint
		}
	}

	return dependencies, nil
}
