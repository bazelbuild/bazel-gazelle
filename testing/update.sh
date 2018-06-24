#! /usr/bin/env bash

set -e

readonly workspace="$(bazel info workspace)"
readonly testlogs="$(bazel info bazel-testlogs)"

cd "${workspace}"

if bazel test --test_output=streamed --cache_test_results=no "//testing:all"; then
  echo "Tests passed. Your goldens are up to date. Doing nothing."
  exit
fi

while IFS= read -d '' -r o; do
  cd "$(dirname "${o}")"
  unzip outputs.zip

  while IFS= read -d '' -r f; do
    echo "pulling $f"
    cp \
      "${f}" \
      "${workspace}/${f/BUILD.bazel/BUILD.in}"
  done < <(find . -name BUILD.bazel -print0)
done < <(find "${testlogs}/testing/" -name outputs.zip -print0)

echo "Testing with updated goldens..."

cd "${workspace}"
bazel test --test_output=streamed --cache_test_results=no "//testing:all"
