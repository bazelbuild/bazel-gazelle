#! /usr/bin/env bash
set -e
err=0

# Since we will be executing Gazelle from a directory other than this one,
# Gazelle must be an absolute path.
path="$1"; shift
gazelle="$1"; shift
args="$1"

echo "Copying all files in the source directory to a tmpdir to remove symlinks..."

copiedAFile=false
newRoot=$(mktemp -d /tmp/gazelle.XXXXXXX)
while IFS= read -d '' -r f; do
  if [[ -f "${f}" ]]; then
    copiedAFile=true
    destination="${newRoot}/${f}"
    mkdir -p "$(dirname "${destination}")" || true
    # cat the file into a new location so it isn't a symlink any more
    cat "${f}" > "${destination}"
  fi
done < <(find -L "${path}" -print0)

if [[ $copiedAFile == false ]]; then
  echo "Failed to copy any files"
  # Unlike most errors, this is fatal and terminal. If there were no files
  # copied there is no reason to continue.
  exit 1
fi

echo "Creating fake WORKSPACE file to allow Gazelle to find a root..."
cat > "${newRoot}/WORKSPACE" <<EOF
# Empty WORKSPACE for testing purposes. DO NOT EDIT.
EOF

cd "${newRoot}"

echo "Generating BUILD.bazel files using Gazelle..."

if ! "${gazelle}" "${args}"; then
  err=$?
  echo "Gazelle exited with a non-zero exit code"
fi

echo "Checking generated vs golden BUILD files for differences..."
golden_build_file="$(mktemp)"
find -L . -name "BUILD.in" > "${golden_build_file}"
golden_build_count=$(( $(wc -l < "${golden_build_file}") ))

generated_build_file="$(mktemp)"
find -L . -name "BUILD.bazel" > "${generated_build_file}"
generated_build_count=$(( $(wc -l < "${generated_build_file}") ))

if (( golden_build_count != generated_build_count )); then
  cat <<EOF
ERROR: The number of generated BUILD files ($generated_build_count) is not
equal to the number of golden build files ($golden_build_count).
EOF
  echo "Golden:"
  cat "${golden_build_file}"
  echo "Generated:"
  cat "${generated_build_file}"
  err=1
fi

while IFS= read -d "" -r f; do
  # In case the user wants to use the TEST_UNDECLARED_OUTPUTS_DIR as an
  # assistant to fix this test, copy it in there.
  destination="${TEST_UNDECLARED_OUTPUTS_DIR}/${f}"
  mkdir -p "$(dirname "${destination}")" || true
  # cat the file into a new location so it can never be a symlink.
  cat "${f}" > "${destination}"

  differences="$(mktemp)"
  if ! diff \
      -y \
      --width=160 \
      "${f}" \
      "${f/BUILD.bazel/BUILD.in}" > "${differences}" 2>&1; then
    echo "Differences were found between ${f}" "${f/BUILD.in/BUILD.bazel}"
    # Or true because the differences file may not have been created if one of the source files didn't exist
    cat "${differences}" || true
    err=2
  fi

done < <(find -L . -name "BUILD.bazel" -print0)

echo
exit $err
