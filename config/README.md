# Analysis rules

This directory contains YAML files that define analysis rules.

## Rule format

A named **ruleset** contains a list of rules and some other metadata.

Then each rule contains the keys:

* `when` (string): the condition (all CEL expressions, for now)
* `then` (string): a known result (if any)
* `maybe` (list): possible results
* `not` (list): specific results to exclude
* `with` (map): a map of strings with any extra data to include in the report (all values are CEL expressions, for now)
* `group`: put this rule into a group for exclusions
* `exclusive` (bool): exclude all other results in this group

[//]: # (TODO document metadata)

## Expressions

Currently, all of the `when` and `with` values are expressions, evaluated using Common Expression Language (CEL).

See [../docs/functions.md](../docs/functions.md) files for a list of all the possible CEL functions.

The expressions are compiled and then cached for better performance. The cache can be generated using the command `make warm_cache`.

Currently, this command will change the cache file every time, even if expressions have not changed.
This may be improved in the future, perhaps by making the functions accept explicit CEL program input instead of closing over pointers.
