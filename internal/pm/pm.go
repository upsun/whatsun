// Package pm detects project managers in a code directory.
package pm

import (
	"fmt"
	"io/fs"
)

type PM struct {
	Type        string
	DetectedVia string
}

type List []PM

func (d List) GetSummary() string {
	return fmt.Sprint(d)
}

// Detect looks for evidence of package managers in a directory.
func Detect(fsys fs.FS) (List, error) {
	var uniqPMs = make(map[string]PM)
	for f, candidates := range filePatterns {
		matches, err := fs.Glob(fsys, f)
		if err != nil {
			return nil, err
		}
		if len(matches) > 0 {
			for c := range candidates {
				uniqPMs[c] = PM{Type: c, DetectedVia: matches[0]}
			}
		}
	}
	var pms = make([]PM, len(uniqPMs))
	i := 0
	for _, pm := range uniqPMs {
		pms[i] = pm
		i++
	}

	return pms, nil
}

const (
	maybe      = 10
	definitely = 100
)

const (
	NPM       = "npm"
	PNPM      = "pnpm"
	Yarn      = "yarn"
	Composer  = "composer"
	Pip       = "pip"
	Pipenv    = "pipenv"
	Poetry    = "poetry"
	PEP518    = "pep518"
	Bundler   = "bundler"
	GoModules = "go_modules"
	Cargo     = "cargo"
	Maven     = "maven"
	Gradle    = "gradle"
	Nuget     = "nuget"
	DotNet    = "dotnet"
	Mix       = "mix"
	QuickLisp = "quicklisp"
)

var filePatterns = map[string]map[string]int{
	// JavaScript
	"package-lock.json": {NPM: definitely},
	"yarn.lock":         {Yarn: definitely},
	"pnpm-lock.yaml":    {PNPM: definitely},
	// TODO disambiguate these things using the other files
	"package.json":   {NPM: maybe, PNPM: maybe, Yarn: maybe},
	"node_modules/.": {NPM: maybe, PNPM: maybe, Yarn: maybe},

	// PHP
	"composer.json": {Composer: definitely},
	"composer.lock": {Composer: definitely},

	// Python
	"requirements.txt": {Pip: maybe},
	"Pipfile":          {Pipenv: definitely},
	"Pipfile.lock":     {Pipenv: definitely},
	"pyproject.toml":   {Poetry: maybe, PEP518: maybe},
	"poetry.lock":      {Poetry: definitely},

	// Ruby
	"Gemfile":      {Bundler: definitely},
	"Gemfile.lock": {Bundler: definitely},

	// Go
	"go.mod": {GoModules: definitely},
	"go.sum": {GoModules: definitely},

	// Rust
	"Cargo.toml": {Cargo: definitely},
	"Cargo.lock": {Cargo: definitely},

	// Java
	"pom.xml":           {Maven: definitely},
	"build.gradle":      {Gradle: definitely},
	"build.gradle.kts":  {Gradle: definitely},
	"settings.gradle":   {Gradle: definitely},
	"gradle.properties": {Gradle: definitely},
	"gradlew":           {Gradle: definitely},
	"gradlew.bat":       {Gradle: definitely},

	// .NET
	"global.json": {DotNet: maybe},
	".csproj":     {DotNet: maybe},
	".fsproj":     {DotNet: maybe},

	// Elixir
	"mix.exs":  {Mix: definitely},
	"mix.lock": {Mix: definitely},

	// Lisp
	"quicklisp/.": {QuickLisp: definitely},

	// Ambiguous
	"vendor/.": {Composer: maybe, Bundler: maybe},
}
