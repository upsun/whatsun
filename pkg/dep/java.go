package dep

import (
	"bufio"
	"encoding/xml"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

type javaManager struct {
	fsys fs.FS
	path string

	initOnce sync.Once
	deps     []Dependency
}

func newJavaManager(fsys fs.FS, path string) Manager {
	return &javaManager{
		fsys: fsys,
		path: path,
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
	deps, err := parsePomXML(m.fsys, m.path)
	if err != nil {
		return err
	}
	m.deps = append(m.deps, deps...)
	deps, err = parseBuildGradleGroovy(m.fsys, m.path)
	if err != nil {
		return err
	}
	m.deps = append(m.deps, deps...)
	deps, err = parseBuildGradleKotlin(m.fsys, m.path)
	if err != nil {
		return err
	}
	m.deps = append(m.deps, deps...)
	deps, err = parseBuildSBT(m.fsys, m.path)
	if err != nil {
		return err
	}
	m.deps = append(m.deps, deps...)
	return nil
}

func (m *javaManager) Find(pattern string) []Dependency {
	var deps []Dependency
	for _, dep := range m.deps {
		if wildcard.Match(pattern, dep.Name) {
			deps = append(deps, dep)
		}
	}
	return deps
}

func (m *javaManager) Get(name string) (Dependency, bool) {
	for _, dep := range m.deps {
		if name == dep.Name {
			return dep, true
		}
	}
	return Dependency{}, false
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

var (
	gradleGroovyPatt = regexp.MustCompile(
		`^(?:implementation|compileOnly|runtimeOnly) ['"]?([^'":]+):([^'":]+):([^'"]+)['"]?$`)
	gradleKotlinPatt = regexp.MustCompile(
		`^(?:implementation|compileOnly|runtimeOnly)\(['"]?([^'":]+):([^'":]+):([^'"]+)['"]?\)$`)
)

func parsePomXML(fsys fs.FS, path string) ([]Dependency, error) {
	f, err := fsys.Open(filepath.Join(path, "pom.xml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var project mavenProject
	if err := xml.NewDecoder(f).Decode(&project); err != nil {
		return nil, err
	}
	var deps = make([]Dependency, 0, len(project.Dependencies.Dependency)+1)
	if project.Parent.GroupID != "" {
		deps = append(deps, Dependency{
			Vendor:  project.Parent.GroupID,
			Name:    project.Parent.GroupID + ":" + project.Parent.ArtifactID,
			Version: project.Parent.Version,
		})
	}
	for _, dep := range project.Dependencies.Dependency {
		deps = append(deps, Dependency{
			Vendor:  dep.GroupID,
			Name:    dep.GroupID + ":" + dep.ArtifactID,
			Version: dep.Version,
		})
	}
	return deps, nil
}

func parseBuildGradle(fsys fs.FS, path, filename string, patt *regexp.Regexp) ([]Dependency, error) {
	f, err := fsys.Open(filepath.Join(path, filename))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var deps []Dependency
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matches := patt.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
		if len(matches) > 3 {
			deps = append(deps, Dependency{Vendor: matches[1], Name: matches[1] + ":" + matches[2], Version: matches[3]})
		}
	}
	return deps, nil
}

func parseBuildGradleGroovy(fsys fs.FS, path string) ([]Dependency, error) {
	return parseBuildGradle(fsys, path, "build.gradle", gradleGroovyPatt)
}

func parseBuildGradleKotlin(fsys fs.FS, path string) ([]Dependency, error) {
	return parseBuildGradle(fsys, path, "build.gradle.kts", gradleKotlinPatt)
}

func parseBuildSBT(fsys fs.FS, path string) ([]Dependency, error) {
	f, err := fsys.Open(filepath.Join(path, "build.sbt"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	var currentLine strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// If we find a libraryDependencies line, start collecting
		if strings.Contains(line, "libraryDependencies") {
			currentLine.WriteString(line)

			// If the line ends with a closing parenthesis, we have a complete declaration
			if strings.HasSuffix(line, ")") {
				deps = append(deps, parseSBTDependencies(currentLine.String())...)
				currentLine.Reset()
			}
		} else if currentLine.Len() > 0 {
			// Continue building the current line
			currentLine.WriteString(" " + line)

			// If we reach the end of the declaration
			if strings.HasSuffix(line, ")") {
				deps = append(deps, parseSBTDependencies(currentLine.String())...)
				currentLine.Reset()
			}
		}
	}

	return deps, nil
}

func parseSBTDependencies(line string) []Dependency {
	var deps []Dependency

	// Pattern to match individual dependencies within SBT syntax
	// Matches: "groupId" %% "artifactId" % "version" (Scala dependencies)
	// and: "groupId" % "artifactId" % "version" (Java dependencies)
	scalaDepPattern := regexp.MustCompile(`['"]([^'"]+)['"]\s*%%\s*['"]([^'"]+)['"]\s*%\s*['"]([^'"]+)['"]`)
	javaDepPattern := regexp.MustCompile(`['"]([^'"]+)['"]\s*%\s*['"]([^'"]+)['"]\s*%\s*['"]([^'"]+)['"]`)

	// Find Scala dependencies first
	scalaMatches := scalaDepPattern.FindAllStringSubmatch(line, -1)
	for _, match := range scalaMatches {
		if len(match) == 4 {
			deps = append(deps, Dependency{
				Vendor:  match[1],
				Name:    match[1] + ":" + match[2],
				Version: match[3],
			})
		}
	}

	// Find Java dependencies
	javaMatches := javaDepPattern.FindAllStringSubmatch(line, -1)
	for _, match := range javaMatches {
		if len(match) == 4 {
			deps = append(deps, Dependency{
				Vendor:  match[1],
				Name:    match[1] + ":" + match[2],
				Version: match[3],
			})
		}
	}

	return deps
}
