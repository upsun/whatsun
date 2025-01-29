package pm

const (
	FrameworkDotNet = "dotnet"
	LanguageElixir  = "elixir"
	LanguageGo      = "go"
	LanguageJava    = "java"
	LanguageJS      = "js"
	LanguageLISP    = "lisp"
	LanguagePHP     = "php"
	LanguagePython  = "python"
	LanguageRuby    = "ruby"
	LanguageRust    = "rust"
)

type PackageManager struct {
	Name string

	// Category is usually a Language* or Framework* constant.
	Category string
}

func (pm *PackageManager) String() string {
	return pm.Category + "/" + pm.Name
}

var (
	NPM       = &PackageManager{Name: "npm", Category: LanguageJS}
	PNPM      = &PackageManager{Name: "pnpm", Category: LanguageJS}
	Yarn      = &PackageManager{Name: "yarn", Category: LanguageJS}
	Bun       = &PackageManager{Name: "bun", Category: LanguageJS}
	Composer  = &PackageManager{Name: "composer", Category: LanguagePHP}
	Pip       = &PackageManager{Name: "pip", Category: LanguagePython}
	Pipenv    = &PackageManager{Name: "pipenv", Category: LanguagePython}
	Poetry    = &PackageManager{Name: "poetry", Category: LanguagePython}
	Bundler   = &PackageManager{Name: "bundler", Category: LanguageRuby}
	GoModules = &PackageManager{Name: "gomod", Category: LanguageGo}
	Cargo     = &PackageManager{Name: "cargo", Category: LanguageRust}
	Maven     = &PackageManager{Name: "maven", Category: LanguageJava}
	Gradle    = &PackageManager{Name: "gradle", Category: LanguageJava}
	Nuget     = &PackageManager{Name: "nuget", Category: FrameworkDotNet}
	Mix       = &PackageManager{Name: "mix", Category: LanguageElixir}
	Quicklisp = &PackageManager{Name: "quicklisp", Category: LanguageLISP}
)

// filePatterns is a map of glob patterns to package managers.
//
// Listing just one package manager implies certainty, as in, that it should be
// the only one detected in that category.
var filePatterns = map[string][]*PackageManager{
	"package-lock.json": {NPM},
	"yarn.lock":         {Yarn},
	"pnpm-lock.yaml":    {PNPM},
	"bun.lock":          {Bun},
	"package.json":      {NPM, PNPM, Yarn, Bun},
	"node_modules/.":    {NPM, PNPM, Yarn, Bun},

	"composer.json": {Composer},
	"composer.lock": {Composer},

	"requirements.txt": {Pip},
	"Pipfile":          {Pipenv},
	"Pipfile.lock":     {Pipenv},
	"pyproject.toml":   {Poetry, Pip},
	"poetry.lock":      {Poetry},

	"Gemfile":      {Bundler},
	"Gemfile.lock": {Bundler},

	"go.mod": {GoModules},
	"go.sum": {GoModules},

	"Cargo.toml": {Cargo},
	"Cargo.lock": {Cargo},

	"pom.xml":           {Maven},
	"build.gradle":      {Gradle},
	"build.gradle.kts":  {Gradle},
	"settings.gradle":   {Gradle},
	"gradle.properties": {Gradle},
	"gradlew":           {Gradle},
	"gradlew.bat":       {Gradle},

	"packages.lock.json": {Nuget},

	"mix.exs":  {Mix},
	"mix.lock": {Mix},

	"quicklisp/.": {Quicklisp},

	"vendor/.": {Composer, Bundler},
}
