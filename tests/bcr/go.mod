// This will stop go mod from descending into this directory.
module github.com/bazelbuild/bazel-gazelle/tests/bcr

go 1.19

// Validate go.mod replace directives can be properly used:
replace github.com/bmatcuk/doublestar/v4 => github.com/bmatcuk/doublestar v1.3.4

require github.com/bmatcuk/doublestar/v4 v4.0.0
