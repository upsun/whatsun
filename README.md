# Whatsun

> [!IMPORTANT]
> This project is still at an early, experimental stage. Anything may change for now, including configuration schemas,
> package interfaces, and even the binary or project names.

Whatsun is a tool and library for code analysis.

It aims to:

* Detect the structure of a code project.
* Detect common usage patterns in the project, such as frameworks, build tools and package managers.
* Provide simple configuration for improving those detection rules.
* Invite improvements, by being open source.

The analysis is intended for other developer tools to provide improved features, such as, potentially:

* UI enhancements, e.g. identifying and filtering projects by the frameworks they use.
* Informing an AI model with an intermediate representation of a codebase.
* Automatic configuration and/or documentation (with or without AI).

## Library usage

```
go get github.com/upsun/whatsun
```

## CLI usage

Currently, you can build this as a CLI from a Git clone: `make build`

Then run it with: `what [path]`

Options (these may change):
* `-ignore string`: Comma-separated list of directory paths to ignore, adding to defaults
* `-rulesets string`: A directory containing custom rulesets, replacing the [default ones](config)

## Configuration and contributions

Analysis rules are defined in YAML inside the [config](config) directory. See [docs/rules.md](docs/rules.md) for more information.
