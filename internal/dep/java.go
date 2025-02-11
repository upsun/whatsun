package dep

import (
	"bufio"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

type javaManager struct {
	fsys fs.FS
	path string

	mu   sync.Mutex
	deps map[string]Dependency
}

func newJavaManager(fsys fs.FS, path string) (Manager, error) {
	m := &javaManager{
		fsys: fsys,
		path: path,
		deps: make(map[string]Dependency),
	}
	if err := m.parse(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *javaManager) parse() error {
	eg := errgroup.Group{}
	eg.Go(func() error {
		f, err := m.fsys.Open(filepath.Join(m.path, "pom.xml"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		defer f.Close()
		project, err := parseMavenDependencies(f)
		if err != nil {
			return err
		}
		var deps = make(map[string]Dependency)
		if project.Parent.GroupID != "" {
			deps[project.Parent.GroupID+":"+project.Parent.ArtifactID] = Dependency{
				Vendor:  project.Parent.GroupID,
				Name:    project.Parent.GroupID + ":" + project.Parent.ArtifactID,
				Version: project.Parent.Version,
			}
		}
		for _, dep := range project.Dependencies.Dependency {
			deps[dep.GroupID+":"+dep.ArtifactID] = Dependency{
				Vendor:  dep.GroupID,
				Name:    dep.GroupID + ":" + dep.ArtifactID,
				Version: dep.Version,
			}
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		for k, v := range deps {
			m.deps[k] = v
		}
		return nil
	})
	eg.Go(func() error {
		f, err := m.fsys.Open(filepath.Join(m.path, "build.gradle"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		defer f.Close()

		reqs, err := parseGradleDependencies(f)
		if err != nil {
			return err
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		for k, v := range reqs {
			vnd, parts := "", strings.SplitN(k, ":", 2)
			if len(parts) == 2 {
				vnd = parts[0]
			}
			m.deps[k] = Dependency{Vendor: vnd, Name: k, Version: v}
		}
		return nil
	})
	return eg.Wait()
}

func (m *javaManager) Find(pattern string) ([]Dependency, error) {
	matches, err := matchDependencyKey(m.deps, pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, nil
	}
	var deps = make([]Dependency, len(matches))
	for i, match := range matches {
		deps[i] = m.deps[match]
	}
	return deps, nil
}

func (m *javaManager) Get(name string) (Dependency, bool) {
	dep, ok := m.deps[name]
	return dep, ok
}

type mavenDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

type mavenProject struct {
	Parent       mavenDependency `xml:"parent"`
	Dependencies struct {
		Dependency []mavenDependency `xml:"dependency"`
	} `xml:"dependencies"`
}

func parseMavenDependencies(r io.Reader) (project mavenProject, err error) {
	err = xml.NewDecoder(r).Decode(&project)
	return
}

type GradleDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func parseGradleDependencies(r io.Reader) (map[string]string, error) {
	var deps map[string]string
	scanner := bufio.NewScanner(r)
	deps = make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "implementation") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				dependency := strings.Trim(parts[1], "'")
				parts := strings.Split(dependency, ":")
				if len(parts) == 3 {
					deps[parts[0]+":"+parts[1]] = parts[2]
				}
			}
		}
	}
	return deps, nil
}
