package celfuncs_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/pkg/eval/celfuncs"
)

func TestCEL(t *testing.T) {
	fsys := fstest.MapFS{
		"foo.txt":        &fstest.MapFile{Data: []byte("foo")},
		"subdir/bar.txt": &fstest.MapFile{Data: []byte("bar")},
		"subdir/foo.txt": &fstest.MapFile{Data: []byte("subdir/foo")},
		"invalid.json":   &fstest.MapFile{Data: []byte(`{{}`)},
		"package.json":   &fstest.MapFile{Data: []byte(`{"name": "test", "dependencies": {"express": "github:expressjs/express"}}`)},
		"bun.lock":       &fstest.MapFile{Data: []byte(`{"packages": {"express": ["express@1.0.0"]}}`)},
	}

	env, err := cel.NewEnv(celfuncs.DefaultEnvOptions()...)
	require.NoError(t, err)

	testCases := []struct {
		expr                  string
		expectResult          any
		expectCompileErr      bool
		expectEvalErrIs       error
		expectEvalErrContains string
		path                  string
	}{
		{expr: `1`, path: ".", expectResult: 1},
		{expr: `fs.fileExists("foo.txt")`, path: ".", expectResult: true},
		{expr: `fs.fileExists(123)`, path: ".", expectCompileErr: true},
		{expr: `fs.fileExists("subdir/bar.txt")`, path: ".", expectResult: true},
		{expr: `fs.fileExists("bar.txt")`, path: ".", expectResult: false},
		{expr: `fs.fileExists("bar.txt")`, path: "subdir", expectResult: true},
		{expr: `fs.read("foo.txt")`, path: ".", expectResult: []byte("foo")},
		{expr: `fs.read("bar.txt")`, path: "subdir", expectResult: []byte("bar")},
		{expr: `fs.read("bar.txt")`, path: ".", expectEvalErrIs: fs.ErrNotExist},
		{expr: `path`, path: ".", expectResult: "."},
		{expr: `path`, path: "subdir", expectResult: "subdir"},
		{expr: `fs.glob("*.txt").size()`, path: ".", expectResult: 1},
		{expr: `fs.glob("*.txt").size() == 2`, path: "subdir", expectResult: true},
		{expr: `fs.isDir(".")`, path: ".", expectResult: true},
		{expr: `fs.isDir("subdir")`, path: ".", expectResult: true},
		{expr: `fs.isDir(".")`, path: "subdir", expectResult: true},
		{expr: `fs.isDir("foo.txt")`, path: ".", expectResult: false},
		{expr: `fs.isDir("nonexistent")`, path: ".", expectResult: false},
		{expr: `fs.fileContains("foo.txt", "foo")`, path: ".", expectResult: true},
		{expr: `fs.fileContains("foo.txt", "subdir")`, path: ".", expectResult: false},
		{expr: `fs.fileContains("foo.txt", "subdir")`, path: "subdir", expectResult: true},
		{expr: `jq(fs.read("package.json"), ".name")`, path: ".", expectResult: "test"},
		{expr: `jq(fs.read("invalid.json"), ".test")`, path: ".", expectEvalErrContains: "invalid character"},
		{expr: `fs.depExists("js", "express")`, path: ".", expectResult: true},
		{expr: `fs.depVersion("js", "express")`, path: ".", expectResult: "1.0.0"},
		{expr: `fs.depExists("js", "nextjs")`, path: ".", expectResult: false},
		{expr: `fs.depExists("swift", "test")`, path: ".", expectEvalErrContains: "manager type not supported"},
	}

	for _, tc := range testCases {
		msgAndArgs := []any{"expr %s, path %s", tc.expr, tc.path}

		input := celfuncs.FilesystemInput(fsys, tc.path)

		ast, iss := env.Compile(tc.expr)
		if tc.expectCompileErr {
			assert.Error(t, iss.Err(), msgAndArgs...)
			continue
		}
		require.NoError(t, iss.Err(), msgAndArgs...)

		prg, err := env.Program(ast)
		require.NoError(t, err, msgAndArgs...)

		result, _, err := prg.Eval(input)
		if tc.expectEvalErrIs != nil {
			assert.ErrorIs(t, err, tc.expectEvalErrIs, msgAndArgs...)
		} else if tc.expectEvalErrContains != "" {
			assert.ErrorContains(t, err, tc.expectEvalErrContains, msgAndArgs...)
		} else {
			assert.NoError(t, err, msgAndArgs...)
			if err == nil {
				assert.Equal(t, types.DefaultTypeAdapter.NativeToValue(tc.expectResult), result, msgAndArgs...)
			}
		}
	}
}
