# what

This is an experimental tool (and potentially library) for code analysis, intended to be useful for automatically generating configuration files.

Build: `make build`

Usage: `./what [path]`

Options (these may change):
* `-ignore string`: Comma-separated list of directory paths to ignore, adding to defaults
* `-allocsprofile string`: Write allocations profile to a file
* `-cpuprofile string`: Write CPU profile to a file
* `-heapprofile string`: Write heap profile to a file

Analysis rules are defined in YAML inside the [config](config) directory. See [docs/rules.md](docs/rules.md) for more information.

The Go import path may be set in future depending on this project's eventual home, e.g. perhaps it will be public on GitHub.
