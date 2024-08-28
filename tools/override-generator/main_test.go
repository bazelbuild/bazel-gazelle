package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBzlmodOverride(t *testing.T) {
	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "simple no override",
			give: `load("@bazel_gazelle//:deps.bzl", "go_repository")

			go_repository(
				name = "com_github_apache_thrift",
				build_file_generation = "auto",
				build_file_proto_mode = "default",
				importpath = "github.com/apache/thrift",
				sum = "h1:cMd2aj52n+8VoAtvSvLn4kDC3aZ6IAkBuqWQ2IDu7wo=",
				version = "v0.17.0",
			)`,
			want: "",
		},
		{
			name: "simple override",
			give: `load("@bazel_gazelle//:deps.bzl", "go_repository")

			go_repository(
				name = "com_github_apache_thrift",
				build_extra_args = ["-go_naming_convention_external=go_default_library"],
				build_file_generation = "on",
				build_file_proto_mode = "disable",
				importpath = "github.com/apache/thrift",
				sum = "h1:cMd2aj52n+8VoAtvSvLn4kDC3aZ6IAkBuqWQ2IDu7wo=",
				version = "v0.17.0",
			)`,
			want: `go_deps = use_extension("//:extensions.bzl", "go_deps")

			go_deps.gazelle_override(
				build_extra_args = ["-go_naming_convention_external=go_default_library"],
				build_file_generation = "on",
				directives = ["gazelle:proto disable"],
				path = "github.com/apache/thrift",
			)`,
		},
		{
			name: "module override and gazelle",
			give: `load("@bazel_gazelle//:deps.bzl", "go_repository")

			go_repository(
				name = "com_github_bazelbuild_bazel_watcher",
				build_extra_args = ["-go_naming_convention_external=go_default_library"],
				build_file_generation = "off",  # keep
				build_file_proto_mode = "disable",
				importpath = "github.com/bazelbuild/bazel-watcher",
				patch_args = ["-p1"],
				patches = [
					# Remove it after they release this PR https://github.com/bazelbuild/bazel-watcher/pull/627
					"//patches:com_github_bazelbuild_bazel_watcher-go-embed.patch",
				],
				sum = "h1:EfJzkMxJuNBGMVdEvkhiW7pAMwhaegbmAMaFCjLjyTw=",
				version = "v0.23.7",
			)`,
			want: `go_deps = use_extension("//:extensions.bzl", "go_deps")

			go_deps.gazelle_override(
				build_extra_args = ["-go_naming_convention_external=go_default_library"],
				build_file_generation = "off",
				directives = ["gazelle:proto disable"],
				path = "github.com/bazelbuild/bazel-watcher",
			)

			go_deps.module_override(
				patch_strip = 1,
				patches = [
					# Remove it after they release this PR https://github.com/bazelbuild/bazel-watcher/pull/627
					"//patches:com_github_bazelbuild_bazel_watcher-go-embed.patch",
				],
				path = "github.com/bazelbuild/bazel-watcher",
			)`,
		},
		{
			name: "directives and proto args",
			give: `go_repository(
				name = "com_github_clickhouse_clickhouse_go_v2",
				build_directives = [
					"gazelle:resolve go github.com/ClickHouse/clickhouse-go/v2/external @com_github_clickhouse_clickhouse_go_v2//external",
				],
				build_extra_args = ["-go_naming_convention_external=go_default_library"],
				build_file_generation = "on",
				build_file_proto_mode = "disable",
				importpath = "github.com/ClickHouse/clickhouse-go/v2",
				sum = "h1:Nbl/NZwoM6LGJm7smNBgvtdr/rxjlIssSW3eG/Nmb9E=",
				version = "v2.0.12",
			)`,
			want: `go_deps = use_extension("//:extensions.bzl", "go_deps")

			go_deps.gazelle_override(
				build_extra_args = ["-go_naming_convention_external=go_default_library"],
				build_file_generation = "on",
				directives = [
					"gazelle:resolve go github.com/ClickHouse/clickhouse-go/v2/external @com_github_clickhouse_clickhouse_go_v2//external",
					"gazelle:proto disable",
				],
				path = "github.com/ClickHouse/clickhouse-go/v2",
			)`,
		},
		{
			name: "archive overrides",
			give: `go_repository(
				name = "org_golang_x_tools_cmd_goimports",
				build_extra_args = [
					"-go_prefix=golang.org/x/tools",
					"-exclude=**/testdata",
				],
				build_file_generation = "on",
				build_file_proto_mode = "disable",
				importpath = "golang.org/x/tools/cmd/goimports",
				patch_args = ["-p1"],
				strip_prefix = "golang.org/x/tools@v0.0.0-20200512131952-2bc93b1c0c88",
				sha256 = "4a6497e0bf1f19c8089dd02e7ba1351ba787f434d62971ff14fb627e57914939",
				patches = [
					"//patches:org_golang_x_tools_cmd_goimports.patch",
				],
				urls = [
					"https://goproxy.uberinternal.com/golang.org/x/tools/@v/v0.0.0-20200512131952-2bc93b1c0c88.zip",
				],
			)`,
			want: `go_deps = use_extension("//:extensions.bzl", "go_deps")

			go_deps.archive_override(
				patch_strip = 1,
				patches = [
					"//patches:org_golang_x_tools_cmd_goimports.patch",
				],
				path = "golang.org/x/tools/cmd/goimports",
				sha256 = "4a6497e0bf1f19c8089dd02e7ba1351ba787f434d62971ff14fb627e57914939",
				strip_prefix = "golang.org/x/tools@v0.0.0-20200512131952-2bc93b1c0c88",
				urls = [
					"https://goproxy.uberinternal.com/golang.org/x/tools/@v/v0.0.0-20200512131952-2bc93b1c0c88.zip",
				],
			)

			go_deps.gazelle_override(
				build_extra_args = [
					"-go_prefix=golang.org/x/tools",
					"-exclude=**/testdata",
				],
				build_file_generation = "on",
				directives = ["gazelle:proto disable"],
				path = "golang.org/x/tools/cmd/goimports",
			)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := t.TempDir()
			testWorkspace := filepath.Join(w, "WORKSPACE")
			require.NoError(t, os.WriteFile(testWorkspace, []byte(removeTabsAndTrimLines(tt.give)), 0644))

			args := &mainArgs{
				workspace:  testWorkspace,
				outputFile: filepath.Join(w, "output.bzl"),
			}

			require.NoError(t, run(*args, io.Discard))

			if tt.want == "" {
				return
			}

			content, err := os.ReadFile(args.outputFile)
			require.NoError(t, err)
			assertIsEqualContent(t, string(content), tt.want)
		})
	}
}

func assertIsEqualContent(t *testing.T, str1, str2 string) {
	cleanStr1 := removeTabsAndTrimLines(str1)
	cleanStr2 := removeTabsAndTrimLines(str2)

	assert.Equal(t, cleanStr1, cleanStr2)
}

// removeTabsAndTrimLines removes tabs, trims leading and trailing spaces on each line,
// and trims leading and trailing newlines from the entire string.
func removeTabsAndTrimLines(str string) string {
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(strings.ReplaceAll(line, "\t", ""))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
