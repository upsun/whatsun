package dep

import (
	"bufio"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

// bazelParser handles parsing of Bazel build files to extract dependencies
type bazelParser struct {
	fsys fs.FS
	path string
	deps map[string][]Dependency // Keyed by language type (java, python, etc)
}

// BazelDependency represents a Bazel-specific dependency
type BazelDependency struct {
	Target   string // e.g., "//lib:mylib" or "@maven//:com_google_guava"
	Rule     string // e.g., "java_library", "py_library"
	External bool   // true for external dependencies like @maven//
}

// newBazelParser creates a new Bazel dependency parser
func newBazelParser(fsys fs.FS, path string) *bazelParser {
	return &bazelParser{
		fsys: fsys,
		path: path,
		deps: make(map[string][]Dependency),
	}
}

// HasBazelFiles checks if the given path contains Bazel build files
func HasBazelFiles(fsys fs.FS, path string) bool {
	bazelFiles := []string{
		"BUILD",
		"BUILD.bazel",
		"WORKSPACE",
		"WORKSPACE.bazel",
		"MODULE.bazel",
	}

	for _, filename := range bazelFiles {
		if _, err := fsys.Open(filepath.Join(path, filename)); err == nil {
			return true
		}
	}
	return false
}

// ParseBazelDependencies parses Bazel dependencies and returns categorized results
func ParseBazelDependencies(fsys fs.FS, path string) (*bazelParser, error) {
	parser := newBazelParser(fsys, path)
	if err := parser.parse(); err != nil {
		return nil, err
	}
	return parser, nil
}

// GetJavaDeps returns Java dependencies found in Bazel files
func (b *bazelParser) GetJavaDeps() []Dependency {
	return b.deps["java"]
}

// GetPythonDeps returns Python dependencies found in Bazel files
func (b *bazelParser) GetPythonDeps() []Dependency {
	return b.deps["python"]
}

// GetAllDeps returns all dependencies regardless of language
func (b *bazelParser) GetAllDeps() []Dependency {
	var allDeps []Dependency
	for _, langDeps := range b.deps {
		allDeps = append(allDeps, langDeps...)
	}
	return allDeps
}

// FindDeps finds dependencies matching a pattern across all languages
func (b *bazelParser) FindDeps(pattern string) []Dependency {
	var deps []Dependency
	for _, dep := range b.GetAllDeps() {
		if wildcard.Match(pattern, dep.Name) {
			deps = append(deps, dep)
		}
	}
	return deps
}

// parse orchestrates parsing of all Bazel files
func (b *bazelParser) parse() error {
	// Parse BUILD files for target dependencies
	if err := b.parseBuildFiles(); err != nil {
		return err
	}

	// Parse MODULE.bazel for modern Bazel dependencies
	if err := b.parseModuleBazel(); err != nil {
		return err
	}

	// Parse WORKSPACE for legacy external dependencies
	if err := b.parseWorkspace(); err != nil {
		return err
	}

	return nil
}

// Regular expressions for parsing Bazel dependencies
var (
	// Match deps = ["//path:target", "@external//path:target"]
	depsPattern = regexp.MustCompile(`deps\s*=\s*\[(.*?)\]`)

	// Match individual dependency strings
	depStringPattern = regexp.MustCompile(`"([^"]+)"`)

	// Match Java rules
	javaRulePattern = regexp.MustCompile(`(java_library|java_binary|java_test)\s*\(`)

	// Match Python rules
	pythonRulePattern = regexp.MustCompile(`(py_library|py_binary|py_test)\s*\(`)

	// Match external Maven dependencies
	mavenDepPattern = regexp.MustCompile(`@maven//:(.+)`)

	// Match bazel_dep declarations in MODULE.bazel
	bazelDepPattern = regexp.MustCompile(`bazel_dep\s*\(\s*name\s*=\s*"([^"]+)"\s*,\s*version\s*=\s*"([^"]+)"`)
)

// parseBuildFiles parses BUILD and BUILD.bazel files for dependencies
func (b *bazelParser) parseBuildFiles() error {
	buildFiles := []string{"BUILD", "BUILD.bazel"}

	for _, filename := range buildFiles {
		if err := b.parseBuildFile(filename); err != nil {
			// If file doesn't exist, continue to next file
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
	}

	return nil
}

// parseBuildFile parses a single BUILD file
func (b *bazelParser) parseBuildFile(filename string) error {
	f, err := b.fsys.Open(filepath.Join(b.path, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var currentRule string
	var inRule bool
	var ruleContent strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check for start of Java or Python rules
		if javaRulePattern.MatchString(line) {
			currentRule = "java"
			inRule = true
			ruleContent.Reset()
		} else if pythonRulePattern.MatchString(line) {
			currentRule = "python"
			inRule = true
			ruleContent.Reset()
		}

		if inRule {
			ruleContent.WriteString(line + " ")

			// Check for end of rule (closing parenthesis)
			if strings.Contains(line, ")") {
				deps := b.extractDepsFromRule(ruleContent.String(), currentRule)
				b.deps[currentRule] = append(b.deps[currentRule], deps...)
				inRule = false
			}
		}
	}

	return scanner.Err()
}

// extractDepsFromRule extracts dependencies from a rule declaration
func (b *bazelParser) extractDepsFromRule(ruleContent, language string) []Dependency {
	var deps []Dependency

	// Find deps = [...] pattern
	depsMatches := depsPattern.FindStringSubmatch(ruleContent)
	if len(depsMatches) < 2 {
		return deps
	}

	// Extract individual dependency strings
	depStrings := depStringPattern.FindAllStringSubmatch(depsMatches[1], -1)
	for _, match := range depStrings {
		if len(match) < 2 {
			continue
		}

		depTarget := match[1]
		dep := b.parseDependencyTarget(depTarget, language)
		if dep.Name != "" {
			deps = append(deps, dep)
		}
	}

	return deps
}

// parseDependencyTarget parses a dependency target string into a Dependency
func (b *bazelParser) parseDependencyTarget(target, language string) Dependency {
	var dep Dependency

	// Handle Maven dependencies
	if mavenMatches := mavenDepPattern.FindStringSubmatch(target); len(mavenMatches) > 1 {
		mavenCoord := mavenMatches[1]
		// Convert maven coordinate format (com_google_guava_guava) to standard format
		// The format is typically groupId_groupId_..._artifactId or just groupId_artifactId
		parts := strings.Split(mavenCoord, "_")
		if len(parts) >= 2 {
			// For coordinates like org_slf4j_slf4j_api, we need to be smarter about parsing
			// Common patterns:
			// - com_google_guava_guava -> com.google.guava:guava
			// - junit_junit -> junit:junit
			// - org_slf4j_slf4j_api -> org.slf4j:slf4j-api

			// Heuristic: if the last part looks like a repeated group name, treat it differently
			lastPart := parts[len(parts)-1]

			// Check if this follows the pattern where artifact name is constructed from multiple parts
			var groupId, artifactId string
			if len(parts) == 2 {
				// Simple case: group_artifact
				groupId = parts[0]
				artifactId = parts[1]
			} else if len(parts) >= 3 {
				// Complex case: try to determine where group ends and artifact begins
				// Look for repeated patterns or common separators

				// Strategy 1: If last two parts are similar to first parts, it might be group_group_artifact
				switch {
				case len(parts) == 4 && parts[0] == parts[1] && parts[1] == parts[2]:
					// Pattern like com_google_guava_guava
					groupId = strings.Join(parts[:len(parts)-1], ".")
					artifactId = lastPart
				case len(parts) == 4 && parts[1] == parts[2]:
					// Pattern like org_slf4j_slf4j_api
					groupId = strings.Join(parts[:2], ".")
					artifactId = strings.Join(parts[2:], "-")
				default:
					// Default: assume last part is artifact, rest is group
					groupId = strings.Join(parts[:len(parts)-1], ".")
					artifactId = lastPart
				}
			}

			dep.Vendor = groupId
			dep.Name = groupId + ":" + artifactId
		} else {
			dep.Name = mavenCoord
		}
		return dep
	}

	// Handle internal dependencies (//path:target)
	if strings.HasPrefix(target, "//") {
		dep.Name = target
		return dep
	}

	// Handle other external dependencies (@repo//path:target)
	if strings.HasPrefix(target, "@") {
		dep.Name = target
		return dep
	}

	// Handle simple target names
	dep.Name = target
	return dep
}

// parseModuleBazel parses MODULE.bazel for modern Bazel dependencies
func (b *bazelParser) parseModuleBazel() error {
	f, err := b.fsys.Open(filepath.Join(b.path, "MODULE.bazel"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Parse bazel_dep declarations
		if matches := bazelDepPattern.FindStringSubmatch(line); len(matches) > 2 {
			dep := Dependency{
				Name:       matches[1],
				Version:    matches[2],
				Constraint: matches[2],
			}

			// Add to general category for now - could be categorized better with more context
			b.deps["bazel"] = append(b.deps["bazel"], dep)
		}
	}

	return scanner.Err()
}

// parseWorkspace parses WORKSPACE files for legacy external dependencies
func (b *bazelParser) parseWorkspace() error {
	workspaceFiles := []string{"WORKSPACE", "WORKSPACE.bazel"}

	for _, filename := range workspaceFiles {
		f, err := b.fsys.Open(filepath.Join(b.path, filename))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
		defer f.Close()

		// For now, just scan for basic patterns
		// A full WORKSPACE parser would be more complex
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip comments and empty lines
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			// Look for maven_install or other dependency declarations
			// This is a simplified parser - real implementation would need more sophistication
			// Future enhancement: parse maven_install and pip_install declarations
			_ = strings.Contains(line, "maven_install") || strings.Contains(line, "pip_install")
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	return nil
}
