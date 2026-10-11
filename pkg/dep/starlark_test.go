package dep

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStarlarkCalls(t *testing.T) {
	src := stripStarlarkComments(`
load("@rules_java//java:defs.bzl", "java_library")  # load(ignored)

java_library(
    name = "a",
    srcs = glob(["*.java"], exclude = ["x(.java"]),  # ) in a comment
    deps = select({"//cond:x": [":x"], "//conditions:default": []}) + [":y"],
)

s = """java_binary(not = "a call")"""
maven.install(artifacts = ['a:b:1'])
`)
	calls := starlarkCalls(src)
	var names []string
	for _, c := range calls {
		names = append(names, c.name)
	}
	assert.Equal(t, []string{"load", "java_library", "maven.install"}, names)

	args := splitStarlarkArgs(calls[1].args)
	assert.Len(t, args, 3)
	assert.Equal(t, `"a"`, starlarkKwarg(args, "name"))
	assert.Equal(t, []string{"//cond:x", ":x", "//conditions:default", ":y"},
		starlarkStrings(starlarkKwarg(args, "deps")))
	assert.Empty(t, starlarkKwarg(args, "data"))

	assert.Equal(t, []string{"a:b:1"}, starlarkStrings(calls[2].args))
}

func TestStarlarkUnterminated(t *testing.T) {
	// Malformed input must not panic.
	for _, src := range []string{`java_library(deps = ["a"`, `X = [`, `"abc`, `'''`, `f(`, `a\`} {
		_ = starlarkCalls(src)
		_ = starlarkStrings(src)
		_ = starlarkAssignments(src, "X")
		_ = splitStarlarkArgs(src)
	}
}

func TestStarlarkAssignment(t *testing.T) {
	src := `
A = ["a", "b"]
AB = A + [
    "c",  # Comment
] + B
C = "x"
A += ["d"]
E == 1
`
	assert.Equal(t, []string{`["a", "b"]`, `["d"]`}, starlarkAssignments(src, "A"))
	assert.Equal(t, []string{"A + [\n    \"c\",  # Comment\n] + B"}, starlarkAssignments(src, "AB"))
	assert.Equal(t, []string{`"x"`}, starlarkAssignments(src, "C"))
	assert.Empty(t, starlarkAssignments(src, "D"))
	assert.Empty(t, starlarkAssignments(src, "E"), "comparison, not assignment")
	assert.Equal(t, []string{"A", "[\n    \"c\",  # Comment\n]", "B"},
		splitStarlarkTopLevel(starlarkAssignments(src, "AB")[0], '+'))
}

func TestStarlarkTerms(t *testing.T) {
	src := `
A = ["a"]
B = (A + ["b"])
C = D
D = C
A += ["a2"]
B = B + ["b2"]
`
	assert.Equal(t, []string{`["a"]`, `["a2"]`, `["b"]`, `["b2"]`, `["c"]`, "glob([\"x\"])"},
		starlarkTerms(src, `(B + ["c"]) + C + glob(["x"])`))
}

func TestBazelRequirementsLabels(t *testing.T) {
	assert.Equal(t, []string{"//:requirements.txt"}, bazelRequirementsLabels(`"//:requirements.txt"`))
	// Only the keys of a dict are labels, not the platform values.
	assert.Equal(t, []string{"//:req_linux.txt", "//:req_osx.txt"},
		bazelRequirementsLabels(`{"//:req_linux.txt": "linux_*", "//:req_osx.txt": "../osx_*",}`))
}

func TestBazelLabelPath(t *testing.T) {
	cases := []struct {
		label, path string
		ok          bool
	}{
		{"//:requirements.txt", "requirements.txt", true},
		{"//tools/publish:requirements_linux.txt", "tools/publish/requirements_linux.txt", true},
		{":requirements.txt", "requirements.txt", true},
		{"requirements.txt", "requirements.txt", true},
		{"@other//:requirements.txt", "", false},
		{"//tools:", "", false},
	}
	for _, c := range cases {
		p, ok := bazelLabelPath(c.label)
		assert.Equal(t, c.ok, ok, c.label)
		if c.ok {
			assert.Equal(t, c.path, p, c.label)
		}
	}
}
