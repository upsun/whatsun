package files_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/upsun/whatsun/pkg/files"
)

type digestTestCase struct {
	name        string
	fsys        fs.FS
	expected    *files.Digest
	expectError string
}

var digestTestCases = []digestTestCase{
	{
		name: "symfony basic",
		fsys: fstest.MapFS{
			`composer.json`: &fstest.MapFile{Data: []byte(`{"require": {"symfony/framework-bundle": "^7", "php": "^8.3"}}`)},
			`composer.lock`: &fstest.MapFile{Data: []byte(
				`{"packages": [{"name": "symfony/framework-bundle", "version": "7.2.3"}]}`)},
		},
		expected: &files.Digest{
			Tree: ".\n  composer.json\n  composer.lock",
			Reports: map[string][]files.Report{
				".": {
					{Result: "symfony", Ruleset: "frameworks", Groups: []string{"php", "symfony"}, With: map[string]any{
						"version": "7.2.3",
					}},
					{Result: "composer", Ruleset: "package_managers", Groups: []string{"php"}, With: map[string]any{
						"php_version": "^8.3",
					}},
				},
			},
			SelectedFiles: []files.FileData{
				{Name: "composer.json", Content: `{"require": {"symfony/framework-bundle": "^7", "php": "^8.3"}}`,
					Cleaned: true, Size: 62},
			},
		},
	},
	{
		name: "big",
		fsys: fstest.MapFS{
			".gitignore": &fstest.MapFile{Data: []byte("/git-ignored/\n" +
				"git-ignored-deep/\n" +
				"git-ignored-wildcard*\n",
			)},

			// Definitely Composer.
			`composer.json`: &fstest.MapFile{Data: []byte(`{"require": {"symfony/framework-bundle": "^7", "php": "^8.3"}}`)},
			`composer.lock`: &fstest.MapFile{Data: []byte(
				`{"packages": [{"name": "symfony/framework-bundle", "version": "7.2.3"}]}`)},

			// Ignored due to .gitignore.
			"git-ignored/composer.json":                      &fstest.MapFile{Data: []byte("{}")},
			"a/b/git-ignored-deep/composer.json":             &fstest.MapFile{Data: []byte("{}")},
			"a/b/git-ignored-wildcard-example/composer.json": &fstest.MapFile{Data: []byte("{}")},

			// Ignored due to .gitignore in a subdirectory.
			"x/y/.gitignore":                  &fstest.MapFile{Data: []byte("ignore-subdir/")},
			"x/y/ignore-subdir/composer.json": &fstest.MapFile{Data: []byte("{}")},

			// Ignored due to default ignores.
			"node_modules/composer.lock": &fstest.MapFile{Data: []byte("{}")},

			// Ignored due to argument ignores.
			"arg-ignored/composer.lock": &fstest.MapFile{Data: []byte("{}")},

			// Potentially Composer or perhaps others.
			"vendor":         &fstest.MapFile{Mode: fs.ModeDir},
			"vendor/symfony": &fstest.MapFile{Mode: fs.ModeDir},

			// Definitely NPM.
			"another-app/package.json":      &fstest.MapFile{Data: []byte("{}")},
			"another-app/package-lock.json": &fstest.MapFile{Data: []byte("{}")},

			// Detected without having a package manager.
			"configured-app/.platform.app.yaml": &fstest.MapFile{Data: []byte("name: app")},

			"rake/Rakefile": &fstest.MapFile{},

			// Ambiguous: Bun, NPM, PNPM, or Yarn.
			// No lockfile so generates an error getting the version.
			"ambiguous/package.json": &fstest.MapFile{Data: []byte(`{"dependencies":{"gatsby":"^5.14.1"}}`)},

			// Eleventy: detected via glob.
			"eleventy/eleventy.config.ts": &fstest.MapFile{},

			// Meteor and NPM directory ("conflicting").
			"meteor/.meteor":           &fstest.MapFile{Mode: fs.ModeDir},
			"meteor/.meteor/packages":  &fstest.MapFile{Data: []byte("meteor-base")},
			"meteor/.meteor/versions":  &fstest.MapFile{Data: []byte("meteor-base@1.5.1")},
			"meteor/package-lock.json": &fstest.MapFile{},

			// Additional directories to increase time taken.
			"deep/1/2/3/4/5/composer.json":     &fstest.MapFile{Data: []byte("{}")},
			"deep/a/b/c/d/e/package.json":      &fstest.MapFile{Data: []byte("{}")},
			"deep/a/b/c/d/e/package-lock.json": &fstest.MapFile{Data: []byte("{}")},
		},
		expected: &files.Digest{
			Tree: ".\n  .gitignore\n  a\n    b\n  ambiguous\n    package.json" +
				"\n  another-app\n    package-lock.json\n    package.json" +
				"\n  arg-ignored\n    composer.lock" +
				"\n  composer.json\n  composer.lock" +
				"\n  configured-app\n    .platform.app.yaml" +
				"\n  deep\n    1\n      2\n        3\n          4\n            5\n              composer.json" +
				"\n    a\n      b\n        c\n          d\n            e\n              ... (2 more)" +
				"\n  eleventy\n    eleventy.config.ts" +
				"\n  meteor\n    .meteor\n      packages\n      versions\n    package-lock.json" +
				"\n  rake\n    Rakefile" +
				"\n  x\n    y\n      .gitignore",
			Reports: map[string][]files.Report{
				".": {
					{Result: "symfony", Ruleset: "frameworks", Groups: []string{"php", "symfony"},
						With: map[string]any{"version": "7.2.3"}},
					{Result: "composer", Ruleset: "package_managers", Groups: []string{"php"},
						With: map[string]any{"php_version": "^8.3"}},
				},
				"ambiguous": {{Result: "gatsby", Ruleset: "frameworks", Groups: []string{"js"},
					With: map[string]any{}}},
				"another-app": {{Result: "npm", Ruleset: "package_managers", Groups: []string{"js"},
					With: map[string]any{}}},
				"configured-app": {{Result: "platformsh-app", Ruleset: "build_tools", Groups: []string{"cloud"},
					With: map[string]any{"name": "app"}}},
				"deep/1/2/3/4/5": {{Result: "composer", Ruleset: "package_managers", Groups: []string{"php"},
					With: map[string]any{}}},
				"deep/a/b/c/d/e": {{Result: "npm", Ruleset: "package_managers", Groups: []string{"js"},
					With: map[string]any{}}},
				"eleventy": {{Result: "eleventy", Ruleset: "frameworks", Groups: []string{"js", "static"},
					With: map[string]any{}}},
				"meteor": {
					{Result: "meteor.js", Ruleset: "frameworks", Groups: []string{"js"}, With: map[string]any{"version": "1.5.1"}},
					{Result: "meteor", Ruleset: "package_managers", Groups: []string{"js"}, With: map[string]any{}},
					{Result: "npm", Ruleset: "package_managers", Groups: []string{"js"}, With: map[string]any{}},
				},
				"rake": {{Result: "rake", Ruleset: "build_tools", Groups: []string{"ruby"}, With: map[string]any{}}},
			},
			SelectedFiles: []files.FileData{
				{Name: "ambiguous/package.json", Content: `{"dependencies":{"gatsby":"^5.14.1"}}`,
					Cleaned: true, Size: 37},
				{Name: "composer.json", Content: `{"require": {"symfony/framework-bundle": "^7", "php": "^8.3"}}`,
					Cleaned: true, Size: 62},
				{Name: "deep/1/2/3/4/5/composer.json", Content: "{}", Cleaned: true, Size: 2},
			},
		},
	},
}

func TestDigest_TestFS_ActualRules(t *testing.T) {
	cnf, err := files.DefaultDigestConfig()
	require.NoError(t, err)

	for _, c := range digestTestCases {
		t.Run(c.name, func(t *testing.T) {
			digester, err := files.NewDigester(c.fsys, cnf)
			require.NoError(t, err)
			digest, err := digester.GetDigest(t.Context())
			if c.expectError != "" {
				assert.EqualError(t, err, c.expectError)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, c.expected, digest)
			}
		})
	}
}

func BenchmarkDigest(b *testing.B) {
	cnf, err := files.DefaultDigestConfig()
	require.NoError(b, err)

	b.ReportAllocs()

	for b.Loop() {
		for _, c := range digestTestCases {
			digester, err := files.NewDigester(c.fsys, cnf)
			require.NoError(b, err)
			_, err = digester.GetDigest(b.Context())
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}
