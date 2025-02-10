# Analysis rules

This directory contains YAML files that define analysis rules.

## Rule format

A named **ruleset** contains **rules** (keyed by name) and some other configuration, for example:

```yaml
example-ruleset:
  max_depth: 0 # Stop searching after the top level
  max_nested_depth: 0 # Stop searching if a result is found after this level
  rules:
    example-rule:
      when: file.exists("Dockerfile")
      then: docker
```

Each rule may contain the keys:

| Key       | Type            | Required? | Description                                      |
|-----------|-----------------|:---------:|--------------------------------------------------|
| when      | string          |    yes    | The condition (always a CEL expression, for now) |
| then      | string          |           | A known result (if any)                          |
| maybe     | list of strings |           | Possible results                                 |
| not       | list of strings |           | Specific results to exclude                      |
| with      | map of strings  |           | Extra data to include in the report              |
| group     | string          |           | A group in which to apply exclusions             |
| exclusive | bool            |           | Exclude all other results in this group          |

Rules and rulesets are not applied in any particular order.

## Expressions

Currently, all of the `when` and `with` values are expressions, evaluated using Common Expression Language (CEL).

See [../docs/functions.md](../docs/functions.md) files for a list of all the possible CEL functions.

The expressions are compiled and then cached for better performance. The cache can be generated using the command `make warm_cache`.

Currently, this command will change the cache file every time, even if expressions have not changed.
This may be improved in the future, perhaps by making the functions accept explicit CEL program input instead of closing over pointers.
