package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/dep"
)

func TestBazelHasFiles(t *testing.T) {
	cases := []struct {
		name     string
		files    map[string][]byte
		expected bool
	}{
		{
			name: "has BUILD file",
			files: map[string][]byte{
				"BUILD": []byte("java_library(name = 'lib')"),
			},
			expected: true,
		},
		{
			name: "has BUILD.bazel file",
			files: map[string][]byte{
				"BUILD.bazel": []byte("java_library(name = 'lib')"),
			},
			expected: true,
		},
		{
			name: "has MODULE.bazel file",
			files: map[string][]byte{
				"MODULE.bazel": []byte("module(name = 'test')"),
			},
			expected: true,
		},
		{
			name: "has WORKSPACE file",
			files: map[string][]byte{
				"WORKSPACE": []byte("workspace(name = 'test')"),
			},
			expected: true,
		},
		{
			name: "no Bazel files",
			files: map[string][]byte{
				"build.gradle": []byte("plugins { id 'java' }"),
			},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for filename, content := range c.files {
				fsys[filename] = &fstest.MapFile{Data: content}
			}

			result := dep.HasBazelFiles(fsys, ".")
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestBazelJavaParsingSimple(t *testing.T) {
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
java_library(
    name = "lib",
    deps = [
        "//internal:common",
        "@maven//:com_google_guava_guava",
        "@maven//:junit_junit",
    ],
)

java_binary(
    name = "main", 
    deps = [
        ":lib",
        "@maven//:org_slf4j_slf4j_api",
    ],
)
		`)},
	}

	parser, err := dep.ParseBazelDependencies(fsys, ".")
	require.NoError(t, err)

	javaDeps := parser.GetJavaDeps()

	expectedDeps := []dep.Dependency{
		{Name: "//internal:common"},
		{Name: "com.google.guava:guava"},
		{Name: "junit:junit"},
		{Name: ":lib"},
		{Name: "org.slf4j:slf4j-api"},
	}

	assert.Len(t, javaDeps, len(expectedDeps))

	// Check that all expected dependencies are found
	for _, expected := range expectedDeps {
		found := false
		for _, actual := range javaDeps {
			if actual.Name == expected.Name {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected dependency %s not found", expected.Name)
	}
}

func TestBazelModuleFile(t *testing.T) {
	fsys := fstest.MapFS{
		"MODULE.bazel": {Data: []byte(`
module(name = "my-module", version = "1.0")

bazel_dep(name = "rules_java", version = "7.1.0")
bazel_dep(name = "rules_cc", version = "0.1.1")
bazel_dep(name = "platforms", version = "0.0.11")
		`)},
	}

	parser, err := dep.ParseBazelDependencies(fsys, ".")
	require.NoError(t, err)

	allDeps := parser.GetAllDeps()

	expectedNames := []string{"rules_java", "rules_cc", "platforms"}

	assert.Len(t, allDeps, len(expectedNames))

	for _, expectedName := range expectedNames {
		found := false
		for _, dep := range allDeps {
			if dep.Name == expectedName {
				assert.NotEmpty(t, dep.Version)
				found = true
				break
			}
		}
		assert.True(t, found, "Expected dependency %s not found", expectedName)
	}
}

func TestBazelJavaIntegration(t *testing.T) {
	// Test that Java manager properly integrates Bazel dependencies
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
java_library(
    name = "lib",
    deps = [
        "@maven//:com_google_guava_guava",
    ],
)
		`)},
		"pom.xml": {Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<project>
    <dependencies>
        <dependency>
            <groupId>org.apache.commons</groupId>
            <artifactId>commons-lang3</artifactId>
            <version>3.12.0</version>
        </dependency>
    </dependencies>
</project>
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypeJava, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	// Should have dependencies from both Maven (pom.xml) and Bazel (BUILD)
	allDeps := m.Find("*")

	// Check that we have dependencies from both sources
	hasMaven := false
	hasBazel := false

	for _, dep := range allDeps {
		if dep.Name == "org.apache.commons:commons-lang3" {
			hasMaven = true
		}
		if dep.Name == "com.google.guava:guava" {
			hasBazel = true
		}
	}

	assert.True(t, hasMaven, "Should have Maven dependency from pom.xml")
	assert.True(t, hasBazel, "Should have Bazel dependency from BUILD file")
}

func TestBazelPythonParsingSimple(t *testing.T) {
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
py_library(
    name = "mylib",
    deps = [
        "//internal:utils",
        "@pip//requests",
        "@pip//flask_cors",
    ],
)

py_binary(
    name = "main",
    deps = [
        ":mylib",
        "@pip//click",
    ],
)
		`)},
	}

	parser, err := dep.ParseBazelDependencies(fsys, ".")
	require.NoError(t, err)

	pythonDeps := parser.GetPythonDeps()

	expectedDeps := []dep.Dependency{
		{Name: "//internal:utils"},
		{Name: "requests"},
		{Name: "flask-cors"},
		{Name: ":mylib"},
		{Name: "click"},
	}

	assert.Len(t, pythonDeps, len(expectedDeps))

	// Check that all expected dependencies are found
	for _, expected := range expectedDeps {
		found := false
		for _, actual := range pythonDeps {
			if actual.Name == expected.Name {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected dependency %s not found", expected.Name)
	}
}

func TestBazelPythonIntegration(t *testing.T) {
	// Test that Python manager properly integrates Bazel dependencies
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
py_library(
    name = "lib",
    deps = [
        "@pip//requests",
    ],
)
		`)},
		"requirements.txt": {Data: []byte(`
flask==2.0.1
pytest>=6.0
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypePython, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	// Should have dependencies from both requirements.txt and Bazel (BUILD)
	allDeps := m.Find("*")

	// Check that we have dependencies from both sources
	hasRequirements := false
	hasBazel := false

	for _, dep := range allDeps {
		if dep.Name == "flask" {
			hasRequirements = true
		}
		if dep.Name == "requests" {
			hasBazel = true
		}
	}

	assert.True(t, hasRequirements, "Should have dependency from requirements.txt")
	assert.True(t, hasBazel, "Should have Bazel dependency from BUILD file")
}

func TestBazelGoParsingSimple(t *testing.T) {
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
go_library(
    name = "mylib",
    deps = [
        "//internal:utils",
        "@com_github_gorilla_mux//:mux",
        "@org_golang_x_time//rate",
    ],
)

go_binary(
    name = "main",
    deps = [
        ":mylib",
        "@com_github_sirupsen_logrus//:logrus",
    ],
)
		`)},
	}

	parser, err := dep.ParseBazelDependencies(fsys, ".")
	require.NoError(t, err)

	goDeps := parser.GetGoDeps()

	expectedDeps := []dep.Dependency{
		{Name: "//internal:utils"},
		{Name: "github.com/gorilla/mux"},
		{Name: "golang.org/x/time"},
		{Name: ":mylib"},
		{Name: "github.com/sirupsen/logrus"},
	}

	assert.Len(t, goDeps, len(expectedDeps))

	// Check that all expected dependencies are found
	for _, expected := range expectedDeps {
		found := false
		for _, actual := range goDeps {
			if actual.Name == expected.Name {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected dependency %s not found", expected.Name)
	}
}

func TestBazelGoIntegration(t *testing.T) {
	// Test that Go manager properly integrates Bazel dependencies
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
go_library(
    name = "lib",
    deps = [
        "@com_github_gorilla_mux//:mux",
    ],
)
		`)},
		"go.mod": {Data: []byte(`
module example.com/myproject

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
)
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypeGo, fsys, ".")
	require.NoError(t, err)
	require.NoError(t, m.Init())

	// Should have dependencies from both go.mod and Bazel (BUILD)
	allDeps := m.Find("*")

	// Check that we have dependencies from both sources
	hasGoMod := false
	hasBazel := false

	for _, dep := range allDeps {
		if dep.Name == "github.com/gin-gonic/gin" {
			hasGoMod = true
		}
		if dep.Name == "github.com/gorilla/mux" {
			hasBazel = true
		}
	}

	assert.True(t, hasGoMod, "Should have dependency from go.mod")
	assert.True(t, hasBazel, "Should have Bazel dependency from BUILD file")
}

func TestBazelFindPattern(t *testing.T) {
	fsys := fstest.MapFS{
		"BUILD": {Data: []byte(`
java_library(
    name = "lib",
    deps = [
        "@maven//:com_google_guava_guava",
        "@maven//:com_google_inject_guice",
        "@maven//:junit_junit",
    ],
)
		`)},
	}

	parser, err := dep.ParseBazelDependencies(fsys, ".")
	require.NoError(t, err)

	// Test wildcard pattern matching
	googleDeps := parser.FindDeps("com.google*")

	expectedCount := 2 // guava and inject
	assert.Len(t, googleDeps, expectedCount)

	for _, dep := range googleDeps {
		assert.True(t,
			dep.Name == "com.google.guava:guava" || dep.Name == "com.google.inject:guice",
			"Unexpected dependency: %s", dep.Name)
	}
}
