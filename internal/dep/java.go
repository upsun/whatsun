package dep

import (
	"bufio"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"golang.org/x/sync/errgroup"
)

type javaManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	mu       sync.Mutex
	deps     map[string]Dependency
}

func newJavaManager(fsys fs.FS, path string) Manager {
	return &javaManager{
		fsys: fsys,
		path: path,
		deps: make(map[string]Dependency),
	}
}

func (m *javaManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.parse()
	})
	return err
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

		reqs, err := parseGradleDependencies(f, gradleGroovyPatt)
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
	eg.Go(func() error {
		f, err := m.fsys.Open(filepath.Join(m.path, "build.gradle.kts"))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		defer f.Close()

		reqs, err := parseGradleDependencies(f, gradleKotlinPatt)
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

func (m *javaManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for name, dep := range m.deps {
		if wildcard.Match(pattern, name) {
			deps = append(deps, dep)
		}
	}
	return deps
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

var (
	gradleGroovyPatt = regexp.MustCompile(`^(?:implementation|compileOnly|runtimeOnly) ['"]?([^'":]+):([^'":]+):([^'"]+)['"]?$`)
	gradleKotlinPatt = regexp.MustCompile(`^(?:implementation|compileOnly|runtimeOnly)\(['"]?([^'":]+):([^'":]+):([^'"]+)['"]?\)$`)
)

func parseGradleDependencies(r io.Reader, patt *regexp.Regexp) (map[string]string, error) {
	var deps map[string]string
	scanner := bufio.NewScanner(r)
	deps = make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := patt.FindStringSubmatch(line)
		if len(matches) > 3 {
			deps[matches[1]+":"+matches[2]] = matches[3]
		}
	}
	return deps, nil
}
