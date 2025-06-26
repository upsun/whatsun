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
	requirements map[string]string // constraints from pyproject.toml, requirements.txt, etc.
	resolved     map[string]string // resolved versions from uv.lock
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

func (m *pythonManager) parseFile(
	filename string,
	parseFunc func(io.Reader) (map[string]string, error),
	targetMap *map[string]string,
) error {
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
	if *targetMap == nil {
		*targetMap = make(map[string]string)
	}
	for k, v := range reqs {
		(*targetMap)[k] = v
	}
	return nil
}

func (m *pythonManager) parse() error {
	// Always try to parse both pyproject.toml and uv.lock if present
	if err := m.parseFile("pyproject.toml", parsePyprojectTOML, &m.requirements); err != nil {
		return err
	}
	if err := m.parseFile("uv.lock", parseUvLock, &m.resolved); err != nil {
		return err
	}
	// Fallbacks for constraints if pyproject.toml is missing
	if len(m.requirements) == 0 {
		if err := m.parseFile("requirements.txt", parseRequirementsTXT, &m.requirements); err != nil {
			return err
		}
		if err := m.parseFile("Pipfile", parsePipfile, &m.requirements); err != nil {
			return err
		}
	}
	return nil
}

func (m *pythonManager) Find(pattern string) []Dependency {
	seen := make(map[string]struct{})
	var deps []Dependency
	// First, add all with constraints
	for name, constraint := range m.requirements {
		if wildcard.Match(pattern, name) {
			deps = append(deps, Dependency{
				Name:       name,
				Constraint: constraint,
				Version:    m.resolved[name],
			})
			seen[name] = struct{}{}
		}
	}
	// Then, add any resolved-only deps not already included
	for name, version := range m.resolved {
		if _, already := seen[name]; !already && wildcard.Match(pattern, name) {
			deps = append(deps, Dependency{
				Name:    name,
				Version: version,
			})
		}
	}
	return deps
}

func (m *pythonManager) Get(name string) (Dependency, bool) {
	constraint, hasConstraint := m.requirements[name]
	version, hasVersion := m.resolved[name]
	if !hasConstraint && !hasVersion {
		return Dependency{}, false
	}
	return Dependency{
		Name:       name,
		Constraint: constraint,
		Version:    version,
	}, true
}

var pipPattern = regexp.MustCompile(`^([\w\-\.]+(?:\[[^\]]+\])?)(.*)$`)

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
			if str, ok := v["version"].(string); ok {
				dependencies[pkg] = str
			}
		default:
			return nil, fmt.Errorf("unrecognized poetry version type %T: %v", version, version)
		}
	}

	// Handle PEP 508/621 dependencies (pip and Poetry 2.0)
	for _, dep := range pyProject.Project.Dependencies {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		matches := pipPattern.FindStringSubmatch(dep)
		if len(matches) > 1 {
			packageName := matches[1]
			versionConstraint := ""
			if len(matches) > 2 {
				versionConstraint = matches[2]
			}
			if packageName != "" {
				dependencies[packageName] = versionConstraint
			}
		}
	}

	return dependencies, nil
}

// parseUvLock parses a uv.lock TOML file and extracts dependencies
func parseUvLock(r io.Reader) (map[string]string, error) {
	type UvPackage struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	}
	type UvLock struct {
		Packages []UvPackage `toml:"package"`
	}

	var lock UvLock
	_, err := toml.NewDecoder(r).Decode(&lock)
	if err != nil {
		return nil, err
	}
	dependencies := make(map[string]string)
	for _, pkg := range lock.Packages {
		dependencies[pkg.Name] = pkg.Version
	}
	return dependencies, nil
}
