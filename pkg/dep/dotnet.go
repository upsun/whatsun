package dep

import (
	"encoding/xml"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

type dotnetManager struct {
	fsys fs.FS
	path string

	csprojFiles []csprojFile
	lockFile    *packagesLock

	initOnce sync.Once
}

type csprojFile struct {
	XMLName       xml.Name `xml:"Project"`
	PropertyGroup []struct {
		TargetFramework string `xml:"TargetFramework"`
	} `xml:"PropertyGroup"`
	ItemGroups []struct {
		PackageReferences []struct {
			Include string `xml:"Include,attr"`
			Version string `xml:"Version,attr"`
		} `xml:"PackageReference"`
	} `xml:"ItemGroup"`
}

type packagesLock struct {
	Version int `json:"version"`
	Targets map[string]map[string]struct {
		Type         string            `json:"type"`
		Dependencies map[string]string `json:"dependencies,omitempty"`
	} `json:"targets"`
}

func newDotnetManager(fsys fs.FS, path string) Manager {
	return &dotnetManager{
		fsys: fsys,
		path: path,
	}
}

func (m *dotnetManager) Init() error {
	var err error
	m.initOnce.Do(func() {
		err = m.init()
	})
	return err
}

func (m *dotnetManager) init() error {
	// Find all .csproj files in the directory
	entries, err := fs.ReadDir(m.fsys, m.path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".csproj") {
			var csproj csprojFile
			if err := m.parseCSProj(entry.Name(), &csproj); err != nil {
				// Continue with other files even if one fails
				continue
			}
			m.csprojFiles = append(m.csprojFiles, csproj)
		}
	}

	// Try to parse packages.lock.json if it exists
	if err := parseJSON(m.fsys, m.path, "packages.lock.json", &m.lockFile); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	return nil
}

func (m *dotnetManager) parseCSProj(filename string, dest *csprojFile) error {
	f, err := m.fsys.Open(filepath.Join(m.path, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	if err := xml.NewDecoder(f).Decode(dest); err != nil {
		return err
	}
	return nil
}

func (m *dotnetManager) Get(name string) (Dependency, bool) {
	for _, csproj := range m.csprojFiles {
		for _, itemGroup := range csproj.ItemGroups {
			for _, pkgRef := range itemGroup.PackageReferences {
				if pkgRef.Include == name {
					version := m.getLockedVersion(pkgRef.Include)
					if version == "" && m.isSpecificVersion(pkgRef.Version) {
						version = pkgRef.Version
					} else if version == "" {
						version = ""
					}
					return Dependency{
						Name:       pkgRef.Include,
						Constraint: pkgRef.Version,
						Version:    version,
					}, true
				}
			}
		}
	}
	return Dependency{}, false
}

// isSpecificVersion checks if a version string represents a specific version (not a range)
func (m *dotnetManager) isSpecificVersion(version string) bool {
	if version == "" {
		return false
	}
	// Exclude NuGet range syntax: [1.0.0,2.0.0), (1.0.0,2.0.0], etc.
	if strings.HasPrefix(version, "[") || strings.HasPrefix(version, "(") {
		return false
	}
	// Check if it's a simple version number like "1.0.0" or "9.0.6"
	// Exclude ranges like ">=1.0.0", "^1.0.0", "1.0.0-*", etc.
	return !strings.ContainsAny(version, "><=^~*")
}

func (m *dotnetManager) Find(pattern string) []Dependency {
	var deps []Dependency
	seen := make(map[string]bool)

	for _, csproj := range m.csprojFiles {
		for _, itemGroup := range csproj.ItemGroups {
			for _, pkgRef := range itemGroup.PackageReferences {
				if wildcard.Match(pattern, pkgRef.Include) && !seen[pkgRef.Include] {
					seen[pkgRef.Include] = true
					deps = append(deps, Dependency{
						Name:       pkgRef.Include,
						Constraint: pkgRef.Version,
						Version:    m.getLockedVersion(pkgRef.Include),
					})
				}
			}
		}
	}
	return deps
}

func (m *dotnetManager) getLockedVersion(packageName string) string {
	if m.lockFile == nil {
		return ""
	}

	// Look through all targets for the package
	for _, target := range m.lockFile.Targets {
		// The key format in NuGet lock files is typically "PackageName/Version"
		for key := range target {
			// Extract package name from key (format: "PackageName/Version")
			if strings.Contains(key, "/") {
				parts := strings.SplitN(key, "/", 2)
				if len(parts) == 2 && parts[0] == packageName {
					return parts[1]
				}
			}
		}
	}
	return ""
}
