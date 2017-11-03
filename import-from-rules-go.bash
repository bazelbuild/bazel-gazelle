#!/bin/bash

set -euo pipefail

# Make sure the old Gazelle is already in GOPATH and installed in GOBIN.
rules_go_gazelle=$GOPATH/src/github.com/bazelbuild/rules_go/go/tools/gazelle
if [ ! -d "$rules_go_gazelle" ]; then
  echo "could not find rules_go gazelle at $rules_go_gazelle" >&2
  exit 1
fi
go install github.com/bazelbuild/rules_go/go/tools/gazelle/gazelle

# Remove old files and copy all packages to the new repository.
files_to_copy=(
    BUILD.bazel
    README.rst
    config
    gazelle
    merger
    packages
    resolve
    rules
    testdata
    wspace
)
cd $(dirname "$0")
for i in "${files_to_copy[@]}"; do
  rm -rf "$i"
  cp -r "$rules_go_gazelle/$i" .
done

# Fix import paths.
find . -name '*.go' \
  -exec sh -c 'sed -e "s!github\.com/bazelbuild/rules_go/go/tools/gazelle!github.com/bazelbuild/bazel-gazelle!g" <{} >{}.tmp ;
      mv "{}.tmp" "{}"' \;

# Fix build files.
gazelle -go_prefix github.com/bazelbuild/bazel-gazelle
patch -Nsp1 -i fix-resolve-genrule.patch
patch -Nsp1 -i fix-testdata.patch

# Run tests
go test github.com/bazelbuild/bazel-gazelle/...
bazel test //...

# Rebuild and reinstall gazelle
go install github.com/bazelbuild/bazel-gazelle/gazelle
