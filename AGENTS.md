# AGENTS.md

This document provides guidelines for AI coding agents working on the Whatsun project.

## Project Overview

Whatsun is a tool and library for code analysis that detects project structure, frameworks, build tools, and package managers.

It can produce an efficient "digest" of a repository (`whatsun --digest`), which can be used to give context to an LLM.

Whatsun is configured via rules defined in YAML in the `config/` directory, which use CEL expressions for pattern matching.

## Code Standards

### Go Version and Idioms
- **REQUIRED**: Use Go 1.24+ idioms
- Use `any` instead of `interface{}`
- Follow standard Go conventions for naming, packaging, and code organization
- Use generics where appropriate

### Project Structure
```
whatsun/
├── cmd/                   # Command-line tools
│   ├── whatsun/           # Main CLI binary
│   ├── gen_docs/          # Documentation generator (updates files in "docs")
│   └── warm_cache/        # Cache warming utility (used by the build)
├── pkg/                   # Public library packages
│   ├── dep/               # Dependency detection
│   ├── eval/              # CEL expression evaluation
│   ├── files/             # File operations and analysis
│   └── rules/             # Rule matching and analysis
├── internal/              # Private packages
├── config/                # Rule definitions in CEL format (in YAML files)
└── docs/                  # Documentation
```

### Dependencies
- Core dependencies: CEL-Go, cobra, go-git, gitleaks
- Testing: Go tests with the help of `github.com/stretchr/testify`
- Avoid adding unnecessary dependencies
- Check existing usage patterns before introducing new libraries

## Build, Test, and Development

### Essential Commands

**Build the project:**
```bash
make build
```
This creates the `whatsun` binary with optimized build flags. It also prewarms the cache in `expr.cache`.

**Run tests:**
```bash
make test
```
Runs unit tests with race detection.

**Run linting:**
```bash
make lint
```
Includes both `go mod tidy` validation and golangci-lint checks.

**Check for vulnerabilities:**
```bash
make govulncheck
```

**Generate documentation:**
```bash
make gen_docs
```
This updates the Markdown files in the `docs` directory.

### Development Workflow
1. Run tests: `make test`
2. Run linting: `make lint`
3. Check vulnerabilities: `make govulncheck`
4. Build: `make build`

### Testing Guidelines
- Write tests for new functionality
- Use testdata directories for test fixtures
- Maintain test coverage with `make test-coverage`
- Use `make bench` for performance testing

## Architecture Guidelines

### Rule System
- Rules are defined in YAML files in `config/`
- CEL expressions are used for pattern matching
- Rules support file pattern matching, dependency detection, and content analysis
- Cache CEL expressions for performance (`expr.cache`)

### Package Organization
- `pkg/dep/`: Language-specific dependency detection (Go, JS, Python, etc.)
- `pkg/eval/`: CEL expression evaluation with caching
- `pkg/files/`: File tree operations, digest generation, comments parsing
- `pkg/rules/`: Rule loading, matching, and analysis coordination

### Performance Considerations
- Use caching for expensive operations (CEL compilation, file analysis)
- Leverage goroutines for parallel processing where appropriate
- Profile with `make profile` for optimization opportunities

## Coding Guidelines

### Error Handling
- Return meaningful errors with context
- Use error wrapping with `fmt.Errorf` and `%w` verb
- Handle errors at appropriate levels

### Code Style
- Follow the linter rules (see `.golangci.yml`)
- Use meaningful variable and function names
- Add package-level documentation
- Keep functions focused and testable

### Configuration
- Use YAML for rule definitions
- Support environment variable configuration where appropriate
- Validate configuration inputs

### Git and Dependencies
- Keep `go.mod` tidy (enforced by `make lint`)
- Use semantic versioning for releases
- Write clear commit messages

## Security
- Never commit secrets or sensitive data
- Use secure file operations
- Validate all external inputs
- Run `make govulncheck` regularly

## Documentation
- Update documentation when changing APIs
- Use `make gen_docs` to regenerate function documentation
- Follow Go documentation conventions
- Document complex algorithms and data structures

## AI Agent Specific Notes

When working on this codebase:

1. **Always run tests and linting** after making changes:
   ```bash
   make test lint
   ```

2. **Understand the rule system** before modifying analysis logic - rules in `config/` drive the core functionality

3. **Use existing patterns** - check similar implementations in `pkg/dep/` when adding new language support

4. **Consider performance** - this tool processes large codebases, so optimize for speed and memory usage

5. **Follow the architecture** - keep public APIs in `pkg/`, private code in `internal/`

6. **Update tests** - add or modify tests for any functionality changes

7. **Validate CLI** - test changes using `./whatsun [path]` (after building), on real projects
