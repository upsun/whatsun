# Analysis rules

The [../config](../config) directory contains YAML files that define analysis rules.

All `.yml` files in the directory are read and combined.

## Rule format

A named **ruleset** contains **rules** (keyed by name) and some other configuration, for example:

```yaml
example-ruleset:
  max_depth: 0 # Stop searching after this level
  rules:
    example-rule:
      when: fs.fileExists("Dockerfile")
      then: docker
```

Each rule may contain the keys:

| Key       | Type            | Required? | Description                                                                                  |
|-----------|-----------------|:---------:|----------------------------------------------------------------------------------------------|
| when      | string          |    yes    | The condition (always a CEL expression, for now)                                             |
| then      | string          |           | A known result (if any). This excludes any `maybe` results for the same subject (directory). |
| maybe     | list of strings |           | Possible results (either `then` or `maybe` is required).                                     |
| not       | list of strings |           | Specific results to exclude                                                                  |
| with      | map of strings  |           | Extra data to include in the report (always CEL expressions, for now)                        |
| group     | string          |           | A group in which to apply exclusions                                                         |
| exclusive | bool            |           | Exclude all other `exclusive` or `maybe` results in this group                               |

Rules and rulesets are not applied in any particular order.

## Expressions

Currently, all of the `when` and `with` values are expressions, evaluated using Common Expression Language (CEL).

See [functions.md](functions.md) files for a list of all the possible CEL functions.

The expressions are compiled and then cached for better performance.
