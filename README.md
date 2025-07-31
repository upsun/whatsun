# Whatsun

Whatsun is a tool and library for code analysis that detects project structure, frameworks, build tools, and package managers.

It can produce an efficient "digest" of a repository (`whatsun digest`), which can be used to give context to an LLM.

Analysis rules are defined in YAML in the [config](config) directory. See [docs/rules.md](docs/rules.md) for more information.

## Library usage

```shell
go get github.com/upsun/whatsun
```

## CLI usage

Install the `whatsun` command with:

```shell
go install github.com/upsun/whatsun/cmd/whatsun@latest
```

Then run it with: `whatsun digest [repository]`

Other commands include `analyze` and `tree`. Run `whatsun` to list commands. Show command help with `--help` (`-h`).
