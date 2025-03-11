# WhatSun

This is a tool and library for code analysis, intended to be useful for automatically generating configuration files.

Build: `make build`

Usage: `./what [path]`

Options (these may change):
* `-ignore string`: Comma-separated list of directory paths to ignore, adding to defaults
* `-rulesets string`: A directory containing custom rulesets, replacing the [default ones](config)

Analysis rules are defined in YAML inside the [config](config) directory. See [docs/rules.md](docs/rules.md) for more information.
