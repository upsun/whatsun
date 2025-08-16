package dep

import (
	"bufio"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
)

// Global caches for performance optimization
var (
	// Cache for Maven coordinate parsing results (most expensive operation)
	mavenCoordCache = sync.Map{} // thread-safe map[string]string
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

// GetGoDeps returns Go dependencies found in Bazel files
func (b *bazelParser) GetGoDeps() []Dependency {
	return b.deps["go"]
}

// GetJSDeps returns JavaScript dependencies found in Bazel files
func (b *bazelParser) GetJSDeps() []Dependency {
	return b.deps["js"]
}

// GetWorkspaceDeps returns WORKSPACE dependencies found in Bazel files
func (b *bazelParser) GetWorkspaceDeps() []Dependency {
	return b.deps["workspace"]
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

	// Match Go rules
	goRulePattern = regexp.MustCompile(`(go_library|go_binary|go_test)\s*\(`)

	// Match JavaScript/Node.js rules
	jsRulePattern = regexp.MustCompile(`(js_library|js_binary|js_test|nodejs_binary|nodejs_test)\s*\(`)

	// Match external Maven dependencies
	mavenDepPattern = regexp.MustCompile(`@maven//:(.+)`)

	// Match external pip dependencies
	pipDepPattern = regexp.MustCompile(`@pip//(.+)`)

	// Match external Go dependencies
	goDepPattern = regexp.MustCompile(`@([^/]+)//.*`)

	// Match external npm dependencies
	npmDepPattern = regexp.MustCompile(`@npm//(.+)`)

	// Match bazel_dep declarations in MODULE.bazel
	bazelDepPattern = regexp.MustCompile(`bazel_dep\s*\(\s*name\s*=\s*"([^"]+)"\s*,\s*version\s*=\s*"([^"]+)"`)

	// Match WORKSPACE dependency declarations
	mavenInstallPattern  = regexp.MustCompile(`maven_install\s*\(`)
	httpArchivePattern   = regexp.MustCompile(`http_archive\s*\(`)
	gitRepositoryPattern = regexp.MustCompile(`git_repository\s*\(`)

	// Match name and version in WORKSPACE declarations
	namePattern    = regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
	versionPattern = regexp.MustCompile(`version\s*=\s*"([^"]+)"`)
	tagPattern     = regexp.MustCompile(`tag\s*=\s*"([^"]+)"`)
	commitPattern  = regexp.MustCompile(`commit\s*=\s*"([^"]+)"`)
)

// parseBuildFiles parses BUILD and BUILD.bazel files for dependencies
func (b *bazelParser) parseBuildFiles() error {
	buildFiles := []string{"BUILD", "BUILD.bazel"}

	// Optimize by checking file existence first to avoid unnecessary I/O
	var existingFiles []string
	for _, filename := range buildFiles {
		if _, err := b.fsys.Open(filepath.Join(b.path, filename)); err == nil {
			existingFiles = append(existingFiles, filename)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	// Parse only existing files
	for _, filename := range existingFiles {
		if err := b.parseBuildFile(filename); err != nil {
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

		// Check for start of language-specific rules
		switch {
		case javaRulePattern.MatchString(line):
			currentRule = "java"
			inRule = true
			ruleContent.Reset()
		case pythonRulePattern.MatchString(line):
			currentRule = "python"
			inRule = true
			ruleContent.Reset()
		case goRulePattern.MatchString(line):
			currentRule = "go"
			inRule = true
			ruleContent.Reset()
		case jsRulePattern.MatchString(line):
			currentRule = "js"
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

	// Pre-allocate slice for better performance
	deps = make([]Dependency, 0, len(depStrings))

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
		dep.Name = b.parseMavenCoordinate(mavenCoord)
		if dep.Name != "" {
			// Extract vendor from coordinate if possible
			if colonIdx := strings.Index(dep.Name, ":"); colonIdx > 0 {
				dep.Vendor = dep.Name[:colonIdx]
			}
		}
		return dep
	}

	// Handle pip dependencies
	if pipMatches := pipDepPattern.FindStringSubmatch(target); len(pipMatches) > 1 {
		pipPackage := pipMatches[1]
		// Convert pip package format to standard Python package name
		// Common patterns: @pip//package_name, @pip//package_name_extra
		dep.Name = strings.ReplaceAll(pipPackage, "_", "-")
		return dep
	}

	// Handle npm dependencies
	if npmMatches := npmDepPattern.FindStringSubmatch(target); len(npmMatches) > 1 {
		npmPackage := npmMatches[1]
		// Convert npm package format to standard package name
		// Common patterns: @npm//package_name, @npm//@scope/package_name
		if strings.HasPrefix(npmPackage, "@") {
			// Handle scoped packages like @npm//@angular/core -> @angular/core
			dep.Name = npmPackage
		} else {
			// Handle regular packages like @npm//lodash -> lodash
			dep.Name = strings.ReplaceAll(npmPackage, "_", "-")
		}
		return dep
	}

	// Handle Go dependencies
	if language == "go" {
		if goMatches := goDepPattern.FindStringSubmatch(target); len(goMatches) > 1 {
			// For Go, external dependencies are typically like @com_github_gorilla_mux//
			// Convert to Go module format: github.com/gorilla/mux
			repoName := goMatches[1]
			// Convert underscores to slashes and dots appropriately
			switch {
			case strings.HasPrefix(repoName, "com_github_"):
				// Handle github.com repositories
				parts := strings.Split(repoName, "_")
				if len(parts) >= 3 {
					dep.Name = "github.com/" + strings.Join(parts[2:], "/")
				} else {
					dep.Name = repoName
				}
			case strings.HasPrefix(repoName, "org_golang_x_"):
				// Handle golang.org/x repositories
				parts := strings.Split(repoName, "_")
				if len(parts) >= 4 {
					dep.Name = "golang.org/x/" + strings.Join(parts[3:], "/")
				} else {
					dep.Name = repoName
				}
			default:
				// Generic conversion: replace underscores with dots/slashes
				dep.Name = strings.ReplaceAll(repoName, "_", ".")
			}
			return dep
		}
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
		if err := b.parseWorkspaceFile(filename); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
	}

	return nil
}

// parseWorkspaceFile parses a single WORKSPACE file
func (b *bazelParser) parseWorkspaceFile(filename string) error {
	f, err := b.fsys.Open(filepath.Join(b.path, filename))
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var currentDeclaration string
	var inDeclaration bool
	var declarationContent strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check for start of dependency declarations
		switch {
		case mavenInstallPattern.MatchString(line):
			currentDeclaration = "maven_install"
			inDeclaration = true
			declarationContent.Reset()
		case httpArchivePattern.MatchString(line):
			currentDeclaration = "http_archive"
			inDeclaration = true
			declarationContent.Reset()
		case gitRepositoryPattern.MatchString(line):
			currentDeclaration = "git_repository"
			inDeclaration = true
			declarationContent.Reset()
		}

		if inDeclaration {
			declarationContent.WriteString(line + " ")

			// Check for end of declaration (closing parenthesis)
			if strings.Contains(line, ")") {
				dep := b.parseWorkspaceDeclaration(declarationContent.String(), currentDeclaration)
				if dep.Name != "" {
					// Add to workspace category
					b.deps["workspace"] = append(b.deps["workspace"], dep)
				}
				inDeclaration = false
			}
		}
	}

	return scanner.Err()
}

// parseWorkspaceDeclaration parses a WORKSPACE dependency declaration
func (b *bazelParser) parseWorkspaceDeclaration(content, declarationType string) Dependency {
	var dep Dependency

	// Extract name
	if nameMatches := namePattern.FindStringSubmatch(content); len(nameMatches) > 1 {
		dep.Name = nameMatches[1]
	}

	// Extract version information based on declaration type
	switch declarationType {
	case "maven_install":
		// For maven_install, we don't get individual dependency info easily
		// This would need more sophisticated parsing of the artifacts list
		dep.Name = "maven_install_" + dep.Name
	case "http_archive":
		// Look for version, tag, or other version indicators
		if versionMatches := versionPattern.FindStringSubmatch(content); len(versionMatches) > 1 {
			dep.Version = versionMatches[1]
			dep.Constraint = versionMatches[1]
		} else if tagMatches := tagPattern.FindStringSubmatch(content); len(tagMatches) > 1 {
			dep.Version = tagMatches[1]
			dep.Constraint = tagMatches[1]
		}
	case "git_repository":
		// Look for tag or commit
		if tagMatches := tagPattern.FindStringSubmatch(content); len(tagMatches) > 1 {
			dep.Version = tagMatches[1]
			dep.Constraint = tagMatches[1]
		} else if commitMatches := commitPattern.FindStringSubmatch(content); len(commitMatches) > 1 {
			dep.Version = commitMatches[1][:8] // Short commit hash
			dep.Constraint = commitMatches[1][:8]
		}
	}

	return dep
}

// parseMavenCoordinate converts Bazel Maven coordinate format to standard Maven coordinate
// with sophisticated heuristics for various patterns
func (b *bazelParser) parseMavenCoordinate(mavenCoord string) string {
	// Check cache first for performance
	if cached, ok := mavenCoordCache.Load(mavenCoord); ok {
		if result, ok := cached.(string); ok {
			return result
		}
	}

	result := b.parseMavenCoordinateUncached(mavenCoord)

	// Cache the result for future use
	mavenCoordCache.Store(mavenCoord, result)

	return result
}

// parseMavenCoordinateUncached performs the actual parsing without caching
func (b *bazelParser) parseMavenCoordinateUncached(mavenCoord string) string {
	// Handle empty or invalid coordinates
	if mavenCoord == "" {
		return ""
	}

	// Split by underscore - this is the standard Bazel convention
	parts := strings.Split(mavenCoord, "_")
	if len(parts) < 2 {
		return mavenCoord // Return as-is if we can't parse it
	}

	// Enhanced pattern recognition for Maven coordinates
	// Common patterns in real-world usage:
	// 1. Simple: group_artifact (junit_junit)
	// 2. Multi-part group: org_springframework_spring_core
	// 3. Repeated components: com_google_guava_guava
	// 4. Complex artifacts: org_slf4j_slf4j_api, io_grpc_grpc_netty_shaded
	// 5. Deep hierarchies: org_apache_commons_commons_lang3

	var groupId, artifactId string

	switch len(parts) {
	case 2:
		// Simple case: group_artifact
		groupId = parts[0]
		artifactId = parts[1]

	case 3:
		// Three parts - need to determine the split
		// Common patterns:
		// - org_junit_jupiter -> org.junit:jupiter
		// - com_fasterxml_jackson -> com.fasterxml:jackson
		groupId = strings.Join(parts[:2], ".")
		artifactId = parts[2]

	case 4:
		// Four parts - most complex cases
		switch {
		case parts[0] == parts[1] && parts[1] == parts[2]:
			// Pattern: com_google_guava_guava -> com.google.guava:guava
			groupId = strings.Join(parts[:3], ".")
			artifactId = parts[3]
		case parts[1] == parts[2]:
			// Pattern: org_slf4j_slf4j_api -> org.slf4j:slf4j-api
			groupId = strings.Join(parts[:2], ".")
			artifactId = strings.Join(parts[2:], "-")
		case b.isKnownGroupPattern(parts):
			// Use known patterns for common libraries
			groupId, artifactId = b.parseKnownPattern(parts)
		default:
			// Default: assume first 3 parts are group, last is artifact
			groupId = strings.Join(parts[:3], ".")
			artifactId = parts[3]
		}

	case 5:
		// Five parts - very complex hierarchies
		switch {
		case b.isKnownGroupPattern(parts):
			groupId, artifactId = b.parseKnownPattern(parts)
		case parts[2] == parts[3]:
			// Pattern like: io_grpc_grpc_netty_shaded -> io.grpc:grpc-netty-shaded
			groupId = strings.Join(parts[:2], ".")
			artifactId = strings.Join(parts[2:], "-")
		default:
			// Default: assume first 4 parts are group, last is artifact
			groupId = strings.Join(parts[:4], ".")
			artifactId = parts[4]
		}

	default:
		// Six or more parts - handle known patterns or default strategy
		if len(parts) >= 6 && b.isKnownGroupPattern(parts) {
			groupId, artifactId = b.parseKnownPattern(parts)
		} else {
			// Conservative default: assume last part is artifact, rest is group
			groupId = strings.Join(parts[:len(parts)-1], ".")
			artifactId = parts[len(parts)-1]
		}
	}

	// Post-processing: normalize common naming conventions
	artifactId = b.normalizeArtifactId(artifactId, groupId)

	return groupId + ":" + artifactId
}

// isKnownGroupPattern checks if the coordinate matches known library patterns
func (b *bazelParser) isKnownGroupPattern(parts []string) bool {
	if len(parts) < 3 {
		return false
	}

	// Check for well-known library patterns
	coordinate := strings.Join(parts, "_")

	// Spring Framework patterns
	if strings.HasPrefix(coordinate, "org_springframework_") {
		return true
	}

	// Apache Commons patterns
	if strings.HasPrefix(coordinate, "org_apache_commons_") {
		return true
	}

	// Jackson patterns
	if strings.HasPrefix(coordinate, "com_fasterxml_jackson_") {
		return true
	}

	// gRPC patterns
	if strings.HasPrefix(coordinate, "io_grpc_") {
		return true
	}

	// Netty patterns
	if strings.HasPrefix(coordinate, "io_netty_") {
		return true
	}

	return false
}

// parseKnownPattern handles specific known library patterns
func (b *bazelParser) parseKnownPattern(parts []string) (string, string) {
	coordinate := strings.Join(parts, "_")

	// Spring Framework: org_springframework_spring_* -> org.springframework:spring-*
	if strings.HasPrefix(coordinate, "org_springframework_spring_") {
		return "org.springframework", strings.Join(parts[2:], "-")
	}

	// Apache Commons: org_apache_commons_commons_* -> org.apache.commons:commons-*
	if strings.HasPrefix(coordinate, "org_apache_commons_commons_") {
		return "org.apache.commons", strings.Join(parts[3:], "-")
	}

	// Jackson: com_fasterxml_jackson_* -> com.fasterxml.jackson.*:jackson-*
	if strings.HasPrefix(coordinate, "com_fasterxml_jackson_") {
		if len(parts) >= 4 {
			groupId := strings.Join(parts[:4], ".")
			artifactId := strings.Join(parts[2:], "-")
			return groupId, artifactId
		}
	}

	// gRPC: io_grpc_grpc_* -> io.grpc:grpc-*
	if strings.HasPrefix(coordinate, "io_grpc_grpc_") {
		return "io.grpc", strings.Join(parts[2:], "-")
	}

	// Netty: io_netty_netty_* -> io.netty:netty-*
	if strings.HasPrefix(coordinate, "io_netty_netty_") {
		return "io.netty", strings.Join(parts[2:], "-")
	}

	// Default fallback
	return strings.Join(parts[:len(parts)-1], "."), parts[len(parts)-1]
}

// normalizeArtifactId applies common normalization rules to artifact IDs
func (b *bazelParser) normalizeArtifactId(artifactId, groupId string) string {
	// No changes needed for most cases, but could add rules here
	// For example, converting underscores to hyphens in artifact names
	// when they're clearly meant to be hyphens

	// Some artifacts use underscores where hyphens are more standard
	// But we need to be conservative to avoid breaking valid cases

	return artifactId
}

// ClearBazelCaches clears all Bazel-related caches to free memory
// This can be called periodically in long-running applications
func ClearBazelCaches() {
	mavenCoordCache = sync.Map{}
}

// GetBazelCacheStats returns statistics about cache usage for monitoring
func GetBazelCacheStats() map[string]int {
	stats := make(map[string]int)

	// Count Maven coordinate cache entries
	mavenCount := 0
	mavenCoordCache.Range(func(_, _ any) bool {
		mavenCount++
		return true
	})
	stats["maven_coordinates"] = mavenCount

	return stats
}
