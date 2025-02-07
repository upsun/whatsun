# Analysis rules

This directory contains YAML files that define analysis rules.

## Rule format

A named **ruleset** contains a list of rules and some other metadata.

Then each rule contains the keys:

* `when` (string): the condition
* `then` (string): a known result (if any)
* `maybe` (list): possible results
* `not` (list): specific results to exclude
* `group`: put this rule into a group for exclusions
* `exclusive` (bool): exclude all other results in this group

[//]: # (TODO document metadata)

## Expressions

Currently, all of the `when` conditions are evaluated using Common Expression Language (CEL).

See [internal/eval/celfuncs](../internal/eval/celfuncs) files for a list of the functions.

[//]: # (TODO document CEL functions)

The expressions are compiled and then cached for better performance. The cache can be generated using the command `make warm_cache`.

Currently, this command will change the cache file every time, even if expressions have not changed.
This may be improved in the future, perhaps by making the functions accept explicit CEL program input instead of closing over pointers.
