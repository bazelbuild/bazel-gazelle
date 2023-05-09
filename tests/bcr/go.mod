// This will stop go mod from descending into this directory.
module github.com/bazelbuild/bazel-gazelle/tests/bcr

go 1.19

// Validate go.mod replace directives can be properly used:
replace github.com/bmatcuk/doublestar/v4 => github.com/bmatcuk/doublestar v1.3.4

require (
	github.com/bmatcuk/doublestar/v4 v4.0.0
	github.com/google/safetext v0.0.0-20220905092116-b49f7bc46da2
	github.com/DataDog/sketches-go v1.4.1
)
