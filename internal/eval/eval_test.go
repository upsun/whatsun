package eval_test

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/eval"
	"what/internal/eval/celfuncs"
)

//go:embed testdata
var testdata embed.FS

//go:embed testdata/expr.cache
var exprCache []byte

func TestEval(t *testing.T) {
	// The embed.FS needs to be translated to real temporary files to support
	// renaming and deletion within the tests.
	testDir := t.TempDir()
	fsys := os.DirFS(testDir)
	copyTestFile(t, "composer_.json", testDir, "composer.json")
	copyTestFile(t, "composer_.lock", testDir, "composer.lock")
	copyTestFile(t, "foo", testDir, "foo")
	copyTestFile(t, "package_.json", testDir, "package.json")
	require.NoError(t, os.Mkdir(filepath.Join(testDir, "subdir"), 0o700))

	cachePath := filepath.Join("testdata", "expr.cache")

	cache, err := eval.NewFileCacheWithContent(exprCache, cachePath)
	require.NoError(t, err)

	var options []cel.EnvOption
	options = append(options, celfuncs.FilesystemVariable())
	options = append(options, celfuncs.AllFileFunctions()...)
	options = append(options, celfuncs.AllPackageManagerFunctions()...)
	options = append(
		options,
		celfuncs.JQ(),
		celfuncs.YQ(),
		celfuncs.ParseVersion(),
	)

	cnf := &eval.Config{EnvOptions: options, Cache: cache}

	e, err := eval.NewEvaluator(cnf)
	require.NoError(t, err)

	ev := func(e *eval.Evaluator, expr string) ref.Val {
		val, err := e.Eval(expr, celfuncs.FilesystemInput(fsys, "."))
		require.NoError(t, err)
		return val
	}

	t.Run("fs.fileExists", func(t *testing.T) {
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileExists("foo")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.glob("fo*").size() > 0`))
		assert.Equal(t, types.Bool(false), ev(e, `fs.fileExists("bar")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileExists("package.json")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileExists("subdir")`))

		// Ensure the same expression can run again (probably from cache).
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileExists("foo")`))
	})

	t.Run("fs.fileExists after changes", func(t *testing.T) {
		// Ensure file rename affects the result.
		require.NoError(t, os.Rename(filepath.Join(testDir, "foo"), filepath.Join(testDir, "bar")))
		assert.Equal(t, types.Bool(false), ev(e, `fs.fileExists("foo")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileExists("bar")`))

		// Ensure file deletion affects the result.
		require.NoError(t, os.Remove(filepath.Join(testDir, "bar")))
		assert.Equal(t, types.Bool(false), ev(e, `fs.fileExists("bar")`))
	})

	t.Run("fs.fileContains", func(t *testing.T) {
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileContains("package.json", "expressjs")`))
		assert.Equal(t, types.Bool(false), ev(e, `fs.fileContains("package.json", "next")`))

		_, err := e.Eval(`fs.fileContains("nonexistent.json", "test")`, celfuncs.FilesystemInput(fsys, "."))
		assert.ErrorIs(t, err, fs.ErrNotExist)
	})

	t.Run("fs.isDir", func(t *testing.T) {
		assert.Equal(t, types.Bool(false), ev(e, `fs.isDir("foo")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.isDir("subdir")`))
		assert.Equal(t, types.Bool(false), ev(e, `fs.isDir("nonexistent")`))
	})

	t.Run("jq", func(t *testing.T) {
		assert.Equal(t, types.String("github:expressjs/express"),
			ev(e, `jq(fs.read("package.json"), ".dependencies.express")`))
	})

	t.Run("package_managers", func(t *testing.T) {
		assert.Equal(t, types.Bool(false), ev(e, `fs.depExists("php", "drupal/core")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.depExists("php", "symfony/framework-bundle")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.depExists("php", "symfony/*")`))
		assert.Equal(t, types.String("3.0.0"), ev(e, `fs.depVersion("php", "psr/cache")`))
		assert.Equal(t, types.String(""), ev(e, `fs.depVersion("php", "drupal/core")`))
		assert.Equal(t, types.Bool(true), ev(e, `fs.depExists("js", "express")`))
	})

	t.Run("parseVersion", func(t *testing.T) {
		assert.Equal(t, types.String("v7.2.3"), ev(e, `fs.depVersion("php", "symfony/framework-bundle")`))
		assert.Equal(t, types.String("7"), ev(e, `parseVersion(fs.depVersion("php", "symfony/framework-bundle")).major`))
		assert.Equal(t, types.String("2"), ev(e, `parseVersion(fs.depVersion("php", "symfony/framework-bundle")).minor`))
		assert.Equal(t, types.String("3"), ev(e, `parseVersion(fs.depVersion("php", "symfony/framework-bundle")).patch`))
	})

	// Ensure the file cache can be saved.
	require.NoError(t, cache.Save())

	// Instantiate everything again to test loading from cache.
	cache, err = eval.NewFileCache(cachePath)
	require.NoError(t, err)
	cnf.Cache = cache
	e, err = eval.NewEvaluator(cnf)
	require.NoError(t, err)

	// Run old and new expressions after cache reload.
	t.Run("after_cache_reload", func(t *testing.T) {
		assert.Equal(t, types.Bool(true), ev(e, `fs.fileExists("package.json")`))
		assert.Equal(t, types.Bool(false), ev(e, `fs.fileExists("bar")`))
	})
}

func copyTestFile(t *testing.T, filename, destDir, destName string) {
	srcFile, err := testdata.Open(filepath.Join("testdata", filename))
	require.NoError(t, err)
	defer srcFile.Close()

	destFile, err := os.Create(filepath.Join(destDir, destName))
	require.NoError(t, err)
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	require.NoError(t, err)
}
