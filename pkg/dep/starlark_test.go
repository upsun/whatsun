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
		_ = starlarkAssignment(src, "X")
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
`
	assert.Equal(t, `["a", "b"]`, starlarkAssignment(src, "A"))
	assert.Equal(t, "A + [\n    \"c\",  # Comment\n] + B", starlarkAssignment(src, "AB"))
	assert.Equal(t, `"x"`, starlarkAssignment(src, "C"))
	assert.Empty(t, starlarkAssignment(src, "D"))
	assert.Equal(t, []string{"A", "[\n    \"c\",  # Comment\n]", "B"},
		splitStarlarkTopLevel(starlarkAssignment(src, "AB"), '+'))
}

func TestStarlarkTerms(t *testing.T) {
	src := `
A = ["a"]
B = (A + ["b"])
C = D
D = C
`
	assert.Equal(t, []string{`["a"]`, `["b"]`, `["c"]`, "glob([\"x\"])"},
		starlarkTerms(src, `(B + ["c"]) + C + glob(["x"])`))
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
