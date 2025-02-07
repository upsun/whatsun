package eval_test

import (
	"embed"
	"io"
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

	evFS := &fsys
	var options []cel.EnvOption
	options = append(options, celfuncs.AllFileFunctions(evFS, nil)...)
	options = append(options, celfuncs.AllPackageManagerFunctions(evFS, nil)...)
	options = append(
		options,
		celfuncs.JSONQueryStringCELFunction(),
		celfuncs.VersionParse(),
	)

	cnf := &eval.Config{EnvOptions: options, Cache: cache}

	e, err := eval.NewEvaluator(cnf)
	require.NoError(t, err)

	ev := func(e *eval.Evaluator, expr string) ref.Val {
		val, err := e.Eval(expr)
		require.NoError(t, err)
		return val
	}

	t.Run("file.exists", func(t *testing.T) {
		assert.Equal(t, types.Bool(true), ev(e, `file.exists("foo")`))
		assert.Equal(t, types.Bool(false), ev(e, `file.exists("bar")`))
		assert.Equal(t, types.Bool(true), ev(e, `file.existsRegex("foo|bar")`))
		assert.Equal(t, types.Bool(true), ev(e, `file.exists("package.json")`))
		assert.Equal(t, types.Bool(true), ev(e, `file.exists("subdir")`))

		// Ensure the same expression can run again (probably from cache).
		assert.Equal(t, types.Bool(true), ev(e, `file.exists("foo")`))
	})

	t.Run("file.exists after changes", func(t *testing.T) {
		// Ensure file rename affects the result.
		require.NoError(t, os.Rename(filepath.Join(testDir, "foo"), filepath.Join(testDir, "bar")))
		assert.Equal(t, types.Bool(false), ev(e, `file.exists("foo")`))
		assert.Equal(t, types.Bool(true), ev(e, `file.exists("bar")`))

		// Ensure file deletion affects the result.
		require.NoError(t, os.Remove(filepath.Join(testDir, "bar")))
		assert.Equal(t, types.Bool(false), ev(e, `file.exists("bar")`))
	})

	t.Run("file.contains", func(t *testing.T) {
		assert.Equal(t, types.Bool(true), ev(e, `file.contains("package.json", "expressjs")`))
		assert.Equal(t, types.Bool(false), ev(e, `file.contains("package.json", "next")`))

		_, err := e.Eval(`file.contains("nonexistent.json", "test")`)
		// TODO why is this not fs.ErrNotExist ?
		assert.ErrorContains(t, err, "no such file")
	})

	t.Run("file.isDir", func(t *testing.T) {
		assert.Equal(t, types.Bool(false), ev(e, `file.isDir("foo")`))
		assert.Equal(t, types.Bool(true), ev(e, `file.isDir("subdir")`))
		assert.Equal(t, types.Bool(false), ev(e, `file.isDir("nonexistent")`))
	})

	t.Run("json", func(t *testing.T) {
		assert.Equal(t, types.String("github:expressjs/express"),
			ev(e, `json.queryString(file.read("package.json"), ".dependencies.express")`))
	})

	t.Run("package_managers", func(t *testing.T) {
		assert.Equal(t, types.Bool(false), ev(e, `composer.requires("drupal/core")`))
		assert.Equal(t, types.Bool(true), ev(e, `composer.requires("symfony/framework-bundle")`))
		assert.Equal(t, types.Bool(true), ev(e, `composer.requires("symfony/*")`))
		assert.Equal(t, types.String("3.0.0"), ev(e, `composer.lockedVersion("psr/cache")`))
		assert.Equal(t, types.String(""), ev(e, `composer.lockedVersion("drupal/core")`))
		assert.Equal(t, types.Bool(true), ev(e, `npm.depends("express")`))
	})

	t.Run("version.parse", func(t *testing.T) {
		assert.Equal(t, types.String("v7.2.3"), ev(e, `composer.lockedVersion("symfony/framework-bundle")`))
		assert.Equal(t, types.String("7"), ev(e, `version.parse(composer.lockedVersion("symfony/framework-bundle")).major`))
		assert.Equal(t, types.String("2"), ev(e, `version.parse(composer.lockedVersion("symfony/framework-bundle")).minor`))
		assert.Equal(t, types.String("3"), ev(e, `version.parse(composer.lockedVersion("symfony/framework-bundle")).patch`))
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
		assert.Equal(t, types.Bool(true), ev(e, `file.exists("package.json")`))
		assert.Equal(t, types.Bool(false), ev(e, `file.exists("bar")`))
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
