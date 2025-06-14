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

```shell
go get github.com/upsun/whatsun
```

## CLI usage

Install the `whatsun` command with:

```shell
GOPRIVATE=github.com/upsun go install github.com/upsun/whatsun/cmd/whatsun@latest
```

Then run it with: `whatsun [path]`

Options (these may change):
* `--help` (`-h`): Display command help
* `--ignore strings`: Paths (or patterns) to ignore, adding to defaults.
* `--digest`: Output a digest of the repository including the file tree, reports, and the contents of selected files.
* `--tree`: Only output a file tree.
* `--rulesets string`: Path to a custom ruleset directory (replacing the [default ones](config)).
* `--filter strings`: Filter the rulesets to ones matching the wildcard pattern(s).
* `--no-meta`: Skip calculating and returning metadata.
* `--json`: Print output in JSON format. Ignored if --digest is set.

## Configuration and contributions

Analysis rules are defined in YAML inside the [config](config) directory. See [docs/rules.md](docs/rules.md) for more information.
