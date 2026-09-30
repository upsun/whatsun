package dep_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/dep"
)

var bazelTestFS = fstest.MapFS{
	"MODULE.bazel": &fstest.MapFile{Data: []byte(`
module(name = "example", version = "1.0")

bazel_dep(name = "rules_jvm_external", version = "6.0")
bazel_dep(name = "rules_python", version = "0.31.0")

maven = use_extension("@rules_jvm_external//:extensions.bzl", "maven")
maven.install(
    artifacts = [
        "org.springframework.boot:spring-boot-starter-web:3.2.1",
        "com.google.guava:guava:33.0.0-jre",  # A comment with "quotes"
        # "org.example:commented-out:1.0",
        maven.artifact(group = "org.slf4j", artifact = "slf4j-api", version = "2.0.9",
                       exclusions = ["com.google.code.gson:gson"]),
    ],
    repositories = ["https://repo1.maven.org/maven2"],
)
maven.artifact(group = "junit", artifact = "junit", version = "4.13.2")
use_repo(maven, "maven")

pip = use_extension("@rules_python//python/extensions:pip.bzl", "pip")
pip.parse(hub_name = "my_deps", python_version = "3.11", requirements_lock = "//:requirements_lock.txt")
`)},
	"go.mod": &fstest.MapFile{Data: []byte(`module example.com/app

go 1.26

require (
	github.com/gorilla/mux v1.8.1
	golang.org/x/net v0.20.0 // indirect
)
`)},

	// A BUILD file at the module root, referencing a module that is indirect in go.mod.
	"BUILD.bazel": &fstest.MapFile{Data: []byte(`go_library(name = "root", deps = ["@org_golang_x_net//http2"])`)},

	"java/BUILD.bazel": &fstest.MapFile{Data: []byte(`
load("@rules_java//java:defs.bzl", "java_library")

java_library(
    name = "app",
    srcs = glob(["src/main/java/**/*.java"]),
    resources = glob(["src/main/resources/**"]),
    deps = [
        ":internal",
        "//other/pkg:lib",
        "@maven//:org_springframework_boot_spring_boot_starter_web",
        "@maven//:com_google_guava_guava",
        "@maven//:org_springframework_spring_context",  # Not declared: resolved by group.
        "@maven//:com_example_unknown_thing",           # Not declared and unknown: skipped.
    ],
    runtime_deps = ["@maven//:junit_junit"],
)
`)},

	"python/BUILD": &fstest.MapFile{Data: []byte(`
load("@my_deps//:requirements.bzl", "requirement")

py_binary(
    name = "main",
    srcs = ["main.py"],
    deps = [
        "@my_deps//django",
        "@pypi//Flask_Cors:pkg",
        requirement("Requests"),
        "//python/lib",
    ],
)
`)},

	"js/BUILD.bazel": &fstest.MapFile{Data: []byte(`
js_library(
    name = "lib",
    srcs = ["index.js"],
    deps = [
        ":node_modules/lodash",
        "//:node_modules/@angular/core",
        "@npm//react",
        "@npm//@types/node",
    ],
)
`)},
	"js/package.json": &fstest.MapFile{Data: []byte(`{"dependencies": {"react": "^18.0.0"}}`)},

	"go/BUILD.bazel": &fstest.MapFile{Data: []byte(`
go_library(
    name = "lib",
    srcs = ["lib.go"],
    deps = [
        "@com_github_gorilla_mux//:mux",
        "@org_golang_x_net//http2",
        "@com_github_pkg_errors//:errors",
        "@io_bazel_rules_go//go/tools/bazel",
    ],
)
`)},

	// A legacy WORKSPACE project in a subdirectory.
	"legacy/WORKSPACE": &fstest.MapFile{Data: []byte(`
load("@rules_jvm_external//:defs.bzl", "maven_install")
load("@bazel_gazelle//:deps.bzl", "go_repository")

BASE = [
    "io.quarkus:quarkus-rest:3.8.0",
]

ARTIFACTS = BASE + [
    "com.google.guava:guava:33.0.0-jre",
]

# Cyclic references are ignored.
CYCLE_A = CYCLE_B
CYCLE_B = CYCLE_A

maven_install(
    name = "deps",
    artifacts = ARTIFACTS + ["junit:junit:4.13.2"] + CYCLE_A,
)

go_repository(
    name = "com_github_go_chi_chi_v5",
    importpath = "github.com/go-chi/chi/v5",
    commit = "abc",
)
`)},
	"legacy/svc/BUILD": &fstest.MapFile{Data: []byte(`
java_binary(name = "svc", main_class = "Main", deps = ["@deps//:io_quarkus_quarkus_rest"])
go_binary(name = "tool", deps = ["@com_github_go_chi_chi_v5//:chi"])
`)},

	"python/requirements.txt": &fstest.MapFile{Data: []byte("Flask_Cors==4.0.0\n")},

	"not-bazel/README.md": &fstest.MapFile{Data: []byte("# Hello")},

	// Directories that happen to be named like Bazel files.
	"dirs-named-bazel/BUILD/README.md":     &fstest.MapFile{Data: []byte("# Hello")},
	"dirs-named-bazel/WORKSPACE/README.md": &fstest.MapFile{Data: []byte("# Hello")},
}

func getBazelTestManager(t *testing.T, managerType, path string) dep.Manager {
	t.Helper()
	m, err := dep.GetManager(managerType, bazelTestFS, path)
	require.NoError(t, err)
	require.NoError(t, m.Init())
	return m
}

func TestBazel_Java(t *testing.T) {
	root := getBazelTestManager(t, dep.ManagerTypeJava, ".")
	assert.ElementsMatch(t, []dep.Dependency{
		{Vendor: "org.springframework.boot", Name: "org.springframework.boot:spring-boot-starter-web",
			Constraint: "3.2.1", Version: "3.2.1", IsDirect: true, ToolName: "bazel"},
		{Vendor: "com.google.guava", Name: "com.google.guava:guava",
			Constraint: "33.0.0-jre", Version: "33.0.0-jre", IsDirect: true, ToolName: "bazel"},
		{Vendor: "org.slf4j", Name: "org.slf4j:slf4j-api",
			Constraint: "2.0.9", Version: "2.0.9", IsDirect: true, ToolName: "bazel"},
		{Vendor: "junit", Name: "junit:junit", Constraint: "4.13.2", Version: "4.13.2", IsDirect: true, ToolName: "bazel"},
	}, root.Find("*"), "exclusions are not dependencies")

	m := getBazelTestManager(t, dep.ManagerTypeJava, "java")
	assert.ElementsMatch(t, []dep.Dependency{
		{Vendor: "org.springframework.boot", Name: "org.springframework.boot:spring-boot-starter-web",
			Constraint: "3.2.1", Version: "3.2.1", IsDirect: true, ToolName: "bazel"},
		{Vendor: "com.google.guava", Name: "com.google.guava:guava",
			Constraint: "33.0.0-jre", Version: "33.0.0-jre", IsDirect: true, ToolName: "bazel"},
		{Vendor: "org.springframework", Name: "org.springframework:spring-context", IsDirect: true, ToolName: "bazel"},
		{Vendor: "junit", Name: "junit:junit", Constraint: "4.13.2", Version: "4.13.2", IsDirect: true, ToolName: "bazel"},
	}, m.Find("*"))
	assert.Len(t, m.Find("org.springframework.boot:*"), 1)

	// Artifact lists built by concatenation.
	legacyRoot := getBazelTestManager(t, dep.ManagerTypeJava, "legacy")
	var names []string
	for _, d := range legacyRoot.Find("*") {
		names = append(names, d.Name)
	}
	assert.Equal(t, []string{"io.quarkus:quarkus-rest", "com.google.guava:guava", "junit:junit"}, names)

	legacy := getBazelTestManager(t, dep.ManagerTypeJava, "legacy/svc")
	assert.Equal(t, []dep.Dependency{
		{Vendor: "io.quarkus", Name: "io.quarkus:quarkus-rest", Constraint: "3.8.0", Version: "3.8.0",
			IsDirect: true, ToolName: "bazel"},
	}, legacy.Find("*"))
}

func TestBazel_Python(t *testing.T) {
	m := getBazelTestManager(t, dep.ManagerTypePython, "python")
	var names []string
	for _, d := range m.Find("*") {
		names = append(names, d.Name)
	}
	// "@pypi//Flask_Cors" is not duplicated, as it is in requirements.txt.
	assert.ElementsMatch(t, []string{"Flask_Cors", "django", "requests"}, names)
	d, ok := m.Get("requests")
	assert.True(t, ok)
	assert.Equal(t, dep.Dependency{Name: "requests", IsDirect: true, ToolName: "bazel"}, d)
}

func TestBazel_JS(t *testing.T) {
	m := getBazelTestManager(t, dep.ManagerTypeJavaScript, "js")
	assert.ElementsMatch(t, []dep.Dependency{
		// From package.json, which takes precedence.
		{Name: "react", Constraint: "^18.0.0", IsDirect: true, ToolName: "npm"},
		{Name: "lodash", IsDirect: true, ToolName: "bazel"},
		{Name: "@angular/core", Vendor: "angular", IsDirect: true, ToolName: "bazel"},
		{Name: "@types/node", Vendor: "types", IsDirect: true, ToolName: "bazel"},
	}, m.Find("*"))
}

func TestBazel_Go(t *testing.T) {
	m := getBazelTestManager(t, dep.ManagerTypeGo, "go")
	assert.ElementsMatch(t, []dep.Dependency{
		{Name: "github.com/gorilla/mux", Version: "v1.8.1", IsDirect: true, ToolName: "bazel"},
		{Name: "golang.org/x/net", Version: "v0.20.0", IsDirect: true, ToolName: "bazel"},
	}, m.Find("*"))

	// Dependencies from the go.mod file itself are not duplicated, and the
	// BUILD file's reference makes an indirect one direct.
	root := getBazelTestManager(t, dep.ManagerTypeGo, ".")
	assert.ElementsMatch(t, []dep.Dependency{
		{Name: "github.com/gorilla/mux", Version: "v1.8.1", IsDirect: true, ToolName: "go"},
		{Name: "golang.org/x/net", Version: "v0.20.0", IsDirect: true, ToolName: "bazel"},
	}, root.Find("*"))
	assert.True(t, hasDep(root, "golang.org/x/net"))

	legacy := getBazelTestManager(t, dep.ManagerTypeGo, "legacy/svc")
	d, ok := legacy.Get("github.com/go-chi/chi/v5")
	assert.True(t, ok)
	assert.Equal(t, dep.Dependency{Name: "github.com/go-chi/chi/v5", Version: "abc", IsDirect: true, ToolName: "bazel"}, d)
}

func TestBazel_NotBazel(t *testing.T) {
	for _, dir := range []string{"not-bazel", "dirs-named-bazel"} {
		for _, managerType := range []string{dep.ManagerTypeJava, dep.ManagerTypePython, dep.ManagerTypeGo} {
			m := getBazelTestManager(t, managerType, dir)
			assert.Empty(t, m.Find("*"), managerType)
		}
	}
}

func hasDep(m dep.Manager, name string) bool {
	_, ok := m.Get(name)
	return ok
}
