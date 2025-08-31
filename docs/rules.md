# Analysis rules

The [../config](../config) directory contains YAML files that define analysis rules.

All `.yml` files in the directory are read and combined.

## Rule format

A named **ruleset** contains **rules** (keyed by name) and some other configuration, for example:

```yaml
example_ruleset:
  rules:
    example-rule:
      when: fs.fileExists("Dockerfile")
      then: docker
```

The rule name has some format restrictions to aid validation; it must only contain lowercase letters `a-z`, numbers
`0-9`, hyphens (`-`), dots (`.`) or underscores (`_`), and the first and last character must be only `a-z` or `0-9`. The
ruleset name has the same restrictions.

Each rule may contain the keys:

| Key    | Type                  | Required? | Description                                                           |
|--------|-----------------------|:---------:|-----------------------------------------------------------------------|
| when   | string                |    yes    | The condition (always a CEL expression, for now)                      |
| then   | list or single string |           | Known result(s) (if any)                                              |
| maybe  | list or single string |           | Possible results (either `then` or `maybe` is required)               |
| with   | map of strings        |           | Extra data to include in the report (always CEL expressions, for now) |
| group  | single string         |           | A group in which `then` results will exclude other `maybe` ones       |
| groups | list or single string |           | Multiple group(s)                                                     |
| ignore | list or single string |           | Directory path(s) to ignore for this rule (in Git's format)           |

Rules and rulesets are not applied in any particular order.

Rules are applied against each directory below the current (or specified) one, except for a brief list of ignored directories.

## Expressions

Currently, all of the `when` and `with` values are expressions, evaluated using Common Expression Language (CEL).

See [functions.md](functions.md) files for a list of all the possible CEL functions.

The expressions are compiled and then cached for better performance.
