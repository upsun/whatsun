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
	dependencies []Dependency
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

func (m *pythonManager) hasFile(filename string) bool {
	_, err := m.fsys.Open(filepath.Join(m.path, filename))
	return err == nil
}

func (m *pythonManager) parseFile(
	filename string,
	parseFunc func(io.Reader, string) ([]Dependency, error),
	toolName string,
) error {
	f, err := m.fsys.Open(filepath.Join(m.path, filename))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()
	deps, err := parseFunc(f, toolName)
	if err != nil {
		return err
	}
	m.dependencies = append(m.dependencies, deps...)
	return nil
}

func (m *pythonManager) mergeResolvedVersions() {
	var merged = make(map[string]Dependency)
	for _, dep := range m.dependencies {
		if existing, found := merged[dep.Name]; found {
			// Merge: prefer constraint info from manifest files, version from lock files
			if dep.Constraint != "" {
				existing.Constraint = dep.Constraint
				existing.IsDirect = dep.IsDirect
				existing.IsDevOnly = dep.IsDevOnly
			}
			if dep.Version != "" {
				existing.Version = dep.Version
			}
			merged[dep.Name] = existing
		} else {
			merged[dep.Name] = dep
		}
	}

	// Convert back to slice
	var i int
	m.dependencies = make([]Dependency, len(merged))
	for _, dep := range merged {
		m.dependencies[i] = dep
		i++
	}
}

func (m *pythonManager) determineTool() string {
	switch {
	case m.hasFile("uv.lock"):
		return "uv"
	case m.hasFile("poetry.lock"):
		return "poetry"
	case m.hasFile("pyproject.toml"):
		// Check if pyproject.toml has specific tool configuration
		if tool := m.detectPyprojectTool(); tool != "" {
			return tool
		}
		return "python" // generic when tool can't be determined
	case m.hasFile("requirements.txt"):
		return "pip"
	case m.hasFile("Pipfile"):
		return "pipenv"
	default:
		return ""
	}
}

func (m *pythonManager) detectPyprojectTool() string {
	f, err := m.fsys.Open(filepath.Join(m.path, "pyproject.toml"))
	if err != nil {
		return ""
	}
	defer f.Close()

	type PyProjectToolDetect struct {
		Tool map[string]toml.Primitive `toml:"tool"`
	}

	var pyProject PyProjectToolDetect
	_, err = toml.NewDecoder(f).Decode(&pyProject)
	if err != nil {
		return ""
	}

	// Check for specific tool sections
	if _, exists := pyProject.Tool["poetry"]; exists {
		return "poetry"
	}
	if _, exists := pyProject.Tool["uv"]; exists {
		return "uv"
	}

	return ""
}

func (m *pythonManager) parse() error {
	switch tool := m.determineTool(); tool {
	case "uv":
		if err := m.parseFile("pyproject.toml", parsePyprojectTOML, tool); err != nil {
			return err
		}
		if err := m.parseFile("uv.lock", parseUvLock, tool); err != nil {
			return err
		}
		m.mergeResolvedVersions()

	case "poetry":
		if err := m.parseFile("pyproject.toml", parsePyprojectTOML, tool); err != nil {
			return err
		}
		if err := m.parseFile("poetry.lock", parsePoetryLock, tool); err != nil {
			return err
		}
		m.mergeResolvedVersions()

	case "python":
		if err := m.parseFile("pyproject.toml", parsePyprojectTOML, tool); err != nil {
			return err
		}

	case "pip":
		if err := m.parseFile("requirements.txt", parseRequirementsTXT, tool); err != nil {
			return err
		}

	case "pipenv":
		if err := m.parseFile("Pipfile", parsePipfile, tool); err != nil {
			return err
		}
	}

	return nil
}

func (m *pythonManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for _, dep := range m.dependencies {
		if wildcard.Match(pattern, dep.Name) {
			deps = append(deps, dep)
		}
	}
	return deps
}

func (m *pythonManager) Get(name string) (Dependency, bool) {
	for _, dep := range m.dependencies {
		if dep.Name == name {
			return dep, true
		}
	}
	return Dependency{}, false
}

var pipPattern = regexp.MustCompile(`^([\w\-\.]+(?:\[[^\]]+\])?)(.*)$`)

// parseRequirementsTXT parses a requirements.txt file
func parseRequirementsTXT(r io.Reader, toolName string) ([]Dependency, error) {
	var dependencies []Dependency
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
			dependencies = append(dependencies, Dependency{
				Name:       packageName,
				Constraint: versionConstraint,
				IsDirect:   true,
				ToolName:   toolName,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dependencies, nil
}

var pipEnvPattern = regexp.MustCompile(`^\s*"?([\w-]+)"?\s*=\s*"([^"]+)"`)

// parsePipfile extracts dependencies from a Pipfile
func parsePipfile(r io.Reader, toolName string) ([]Dependency, error) {
	var dependencies []Dependency

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := pipEnvPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			dependencies = append(dependencies, Dependency{
				Name:       matches[1],
				Constraint: matches[2],
				IsDirect:   true,
				ToolName:   toolName,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return dependencies, nil
}

// parsePyprojectTOML parses pyproject.toml dependencies (Poetry and PEP 621)
func parsePyprojectTOML(r io.Reader, toolName string) ([]Dependency, error) {
	type PyProject struct {
		Project struct {
			Dependencies []string            `toml:"dependencies"`
			Optional     map[string][]string `toml:"optional-dependencies"`
		} `toml:"project"`
		DependencyGroups map[string][]string `toml:"dependency-groups"` // PEP 735
		Tool             struct {
			Poetry struct {
				Dependencies    map[string]any                       `toml:"dependencies"`
				DevDependencies map[string]any                       `toml:"dev-dependencies"` // Legacy Poetry
				Group           map[string]map[string]map[string]any `toml:"group"`            // Modern Poetry groups
			} `toml:"poetry"`
		} `toml:"tool"`
	}

	var pyProject PyProject
	_, err := toml.NewDecoder(r).Decode(&pyProject)
	if err != nil {
		return nil, err
	}

	var dependencies []Dependency

	// Handle Poetry dependencies
	for pkg, version := range pyProject.Tool.Poetry.Dependencies {
		switch v := version.(type) {
		case string:
			dependencies = append(dependencies, Dependency{
				Name:       pkg,
				Constraint: v,
				IsDirect:   true,
				ToolName:   toolName,
			})
		case map[string]any:
			if str, ok := v["version"].(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:       pkg,
					Constraint: str,
					IsDirect:   true,
					ToolName:   toolName,
				})
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
			dependencies = append(dependencies, Dependency{
				Name:       packageName,
				Constraint: versionConstraint,
				IsDirect:   true,
				ToolName:   toolName,
			})
		}
	}

	// Handle PEP 735 dependency groups (uv standard)
	if devGroup, ok := pyProject.DependencyGroups["dev"]; ok {
		for _, dep := range devGroup {
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
				dependencies = append(dependencies, Dependency{
					Name:       packageName,
					Constraint: versionConstraint,
					IsDirect:   true,
					IsDevOnly:  true,
					ToolName:   toolName,
				})
			}
		}
	}

	// Handle Poetry legacy dev dependencies
	for pkg, version := range pyProject.Tool.Poetry.DevDependencies {
		switch v := version.(type) {
		case string:
			dependencies = append(dependencies, Dependency{
				Name:       pkg,
				Constraint: v,
				IsDirect:   true,
				IsDevOnly:  true,
				ToolName:   toolName,
			})
		case map[string]any:
			if str, ok := v["version"].(string); ok {
				dependencies = append(dependencies, Dependency{
					Name:       pkg,
					Constraint: str,
					IsDirect:   true,
					IsDevOnly:  true,
					ToolName:   toolName,
				})
			}
		default:
			return nil, fmt.Errorf("unrecognized poetry dev version type %T: %v", version, version)
		}
	}

	// Handle Poetry groups (modern Poetry)
	if devGroup, ok := pyProject.Tool.Poetry.Group["dev"]; ok {
		if deps, ok := devGroup["dependencies"]; ok {
			for pkg, version := range deps {
				switch v := version.(type) {
				case string:
					dependencies = append(dependencies, Dependency{
						Name:       pkg,
						Constraint: v,
						IsDirect:   true,
						IsDevOnly:  true,
						ToolName:   toolName,
					})
				case map[string]any:
					if str, ok := v["version"].(string); ok {
						dependencies = append(dependencies, Dependency{
							Name:       pkg,
							Constraint: str,
							IsDirect:   true,
							IsDevOnly:  true,
							ToolName:   toolName,
						})
					}
				default:
					return nil, fmt.Errorf("unrecognized poetry group dev version type %T: %v", version, version)
				}
			}
		}
	}

	// Handle PEP 621 optional dependencies (treat "dev" group as dev dependencies)
	if devGroup, ok := pyProject.Project.Optional["dev"]; ok {
		for _, dep := range devGroup {
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
				dependencies = append(dependencies, Dependency{
					Name:       packageName,
					Constraint: versionConstraint,
					IsDirect:   true,
					IsDevOnly:  true,
					ToolName:   toolName,
				})
			}
		}
	}

	return dependencies, nil
}

// parseUvLock parses a uv.lock TOML file and extracts dependencies
func parseUvLock(r io.Reader, toolName string) ([]Dependency, error) {
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
	var dependencies []Dependency
	for _, pkg := range lock.Packages {
		dependencies = append(dependencies, Dependency{
			Name:     pkg.Name,
			Version:  pkg.Version,
			IsDirect: false, // Lock files contain both direct and indirect deps
			ToolName: toolName,
		})
	}
	return dependencies, nil
}

// parsePoetryLock parses a poetry.lock TOML file and extracts dependencies
func parsePoetryLock(r io.Reader, toolName string) ([]Dependency, error) {
	type PoetryPackage struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	}
	type PoetryLock struct {
		Packages []PoetryPackage `toml:"package"`
	}

	var lock PoetryLock
	_, err := toml.NewDecoder(r).Decode(&lock)
	if err != nil {
		return nil, err
	}
	var dependencies []Dependency
	for _, pkg := range lock.Packages {
		dependencies = append(dependencies, Dependency{
			Name:     pkg.Name,
			Version:  pkg.Version,
			IsDirect: false, // Lock files contain both direct and indirect deps
			ToolName: toolName,
		})
	}
	return dependencies, nil
}
