# Bazel Go Repository to Gazelle Overrides Converter

## WARNING: In development. This tool may functionally change and an experimental project.

## Description
This script converts `go_repository` rules to Gazelle `go_deps` overrides to assist in the migration to Bzlmod.

## Usage
Run the script with the following flags:

- `--macro`: Path to a macro file to translate to overrides.
- `--def_name`: Name of the macro's function name that loads the `go_repository` rules.
- `--workspace`: Path to the workspace file, to load translate all rules loaded from the workspace.
- `--output`: Path to the output file.
- `--help`: Show help message.

Only one of `--macro` or `--workspace` should be specified. The `--def_name` is required when `--macro` is specified.

Example:
```
go run main.go --workspace /path/to/WORKSPACE --output /path/to/output.bzl
```

## Output
The script outputs a file containing `go_deps` overrides based on the provided `go_repository` rules.
