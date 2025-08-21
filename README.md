# Whatsun

Whatsun is a command-line tool and Go library for code analysis.

The primary use case is to generate a repository **digest** (a concise, token-efficient summary), which can be used in
an LLM prompt to provide context for code-related tasks.

## Features

### CLI Commands

- **`whatsun digest`** - Generate a repository summary for LLM context, including detected technologies and the contents of key files
- **`whatsun analyze`** - Perform detailed (rule-based) analysis and show detected frameworks, build tools, and package managers
- **`whatsun deps`** - List all dependencies found across the repository with their sources and versions
- **`whatsun tree`** - Display a concise repository file structure

### Core Capabilities

* Multi-language dependency detection (Go, JavaScript, Python, Java, PHP, Ruby, Rust, and more)
* Configurable rules (using CEL expressions), which by default identify frameworks, build tools, and package managers
* Handling of Git excludes (`.gitignore` and `.git/info/exclude` files) when analyzing the repository.
* Fast processing including caching and parallel analysis.
* A digest structure optimized for use in an LLM context, containing:
  - A token-efficient file tree.
  - The reports from rule-based analysis.
  - The contents of automatically selected files (filtered using `.aiignore` and `.aiexclude` files, sanitized using
    secret detection, and truncated).

## Installation

### CLI Tool

```shell
go install github.com/upsun/whatsun/cmd/whatsun@latest
```

### Go Library

```shell
go get github.com/upsun/whatsun
```

## Usage

### Command Line

```shell
# Generate repository digest (using a local file path or a URL)
whatsun digest [repository]

# Analyze project structure
whatsun analyze [repository]

# List all dependencies
whatsun deps [repository]

# Show file tree
whatsun tree [repository]
```

Run `whatsun --help` for detailed command options.

### Configuration

Analysis rules are defined in YAML files in the [config](config) directory using CEL expressions for pattern matching. See [docs/rules.md](docs/rules.md) for rule configuration details.
