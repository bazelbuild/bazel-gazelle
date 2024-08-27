# Code generated by list_repository_tools_srcs.go; DO NOT EDIT.
# regenerate with `go run internal/list_repository_tools_srcs.go -dir $PWD -generate internal/go_repository_tools_srcs.bzl`
GO_REPOSITORY_TOOLS_SRCS = [
    Label("//:BUILD.bazel"),
    Label("//cmd:BUILD.bazel"),
    Label("//cmd/autogazelle:BUILD.bazel"),
    Label("//cmd/autogazelle:client_unix.go"),
    Label("//cmd/autogazelle:main.go"),
    Label("//cmd/autogazelle:server_unix.go"),
    Label("//cmd/fetch_repo:BUILD.bazel"),
    Label("//cmd/fetch_repo:clean.go"),
    Label("//cmd/fetch_repo:copy_tree.go"),
    Label("//cmd/fetch_repo:errorscompat.go"),
    Label("//cmd/fetch_repo:go_mod_download.go"),
    Label("//cmd/fetch_repo:main.go"),
    Label("//cmd/fetch_repo:module.go"),
    Label("//cmd/fetch_repo:path.go"),
    Label("//cmd/fetch_repo:vcs.go"),
    Label("//cmd/gazelle:BUILD.bazel"),
    Label("//cmd/gazelle:diff.go"),
    Label("//cmd/gazelle:fix-update.go"),
    Label("//cmd/gazelle:fix.go"),
    Label("//cmd/gazelle:langs.go"),
    Label("//cmd/gazelle:main.go"),
    Label("//cmd/gazelle:metaresolver.go"),
    Label("//cmd/gazelle:print.go"),
    Label("//cmd/gazelle:profiler.go"),
    Label("//cmd/gazelle:update-repos.go"),
    Label("//cmd/generate_repo_config:BUILD.bazel"),
    Label("//cmd/generate_repo_config:main.go"),
    Label("//cmd/move_labels:BUILD.bazel"),
    Label("//cmd/move_labels:main.go"),
    Label("//config:BUILD.bazel"),
    Label("//config:config.go"),
    Label("//config:constants.go"),
    Label("//convention:BUILD.bazel"),
    Label("//convention:check.go"),
    Label("//convention:config.go"),
    Label("//convention:dir_set.go"),
    Label("//flag:BUILD.bazel"),
    Label("//flag:flag.go"),
    Label("//internal:BUILD.bazel"),
    Label("//internal/bzlmod:BUILD.bazel"),
    Label("//internal/gazellebinarytest:BUILD.bazel"),
    Label("//internal/gazellebinarytest:xlang.go"),
    Label("//internal/generationtest:BUILD.bazel"),
    Label("//internal/language:BUILD.bazel"),
    Label("//internal/language/test_filegroup:BUILD.bazel"),
    Label("//internal/language/test_filegroup:lang.go"),
    Label("//internal/language/test_load_for_packed_rules:BUILD.bazel"),
    Label("//internal/language/test_load_for_packed_rules:lang.go"),
    Label("//internal/language/test_loads_from_flag:BUILD.bazel"),
    Label("//internal/language/test_loads_from_flag:lang.go"),
    Label("//internal:list_repository_tools_srcs.go"),
    Label("//internal/module:BUILD.bazel"),
    Label("//internal/module:module.go"),
    Label("//internal/version:BUILD.bazel"),
    Label("//internal/version:version.go"),
    Label("//internal/wspace:BUILD.bazel"),
    Label("//internal/wspace:finder.go"),
    Label("//label:BUILD.bazel"),
    Label("//label:label.go"),
    Label("//language:BUILD.bazel"),
    Label("//language:base.go"),
    Label("//language/bazel:BUILD.bazel"),
    Label("//language/bazel/visibility:BUILD.bazel"),
    Label("//language/bazel/visibility:config.go"),
    Label("//language/bazel/visibility:lang.go"),
    Label("//language/bazel/visibility:resolve.go"),
    Label("//language/go:BUILD.bazel"),
    Label("//language/go:build_constraints.go"),
    Label("//language/go:config.go"),
    Label("//language/go:constants.go"),
    Label("//language/go:embed.go"),
    Label("//language/go:fileinfo.go"),
    Label("//language/go:fix.go"),
    Label("//language/go/gen_std_package_list:BUILD.bazel"),
    Label("//language/go/gen_std_package_list:gen_std_package_list.go"),
    Label("//language/go:generate.go"),
    Label("//language/go:kinds.go"),
    Label("//language/go:lang.go"),
    Label("//language/go:modules.go"),
    Label("//language/go:package.go"),
    Label("//language/go:resolve.go"),
    Label("//language/go:std_package_list.go"),
    Label("//language/go:stdlib_links.go"),
    Label("//language/go:update.go"),
    Label("//language/go:utils.go"),
    Label("//language/go:work.go"),
    Label("//language:lang.go"),
    Label("//language:lifecycle.go"),
    Label("//language/proto:BUILD.bazel"),
    Label("//language/proto:config.go"),
    Label("//language/proto:constants.go"),
    Label("//language/proto:fileinfo.go"),
    Label("//language/proto:fix.go"),
    Label("//language/proto/gen:BUILD.bazel"),
    Label("//language/proto/gen:gen_known_imports.go"),
    Label("//language/proto:generate.go"),
    Label("//language/proto:kinds.go"),
    Label("//language/proto:known_go_imports.go"),
    Label("//language/proto:known_imports.go"),
    Label("//language/proto:known_proto_imports.go"),
    Label("//language/proto:lang.go"),
    Label("//language/proto:package.go"),
    Label("//language/proto:resolve.go"),
    Label("//language:update.go"),
    Label("//merger:BUILD.bazel"),
    Label("//merger:fix.go"),
    Label("//merger:merger.go"),
    Label("//pathtools:BUILD.bazel"),
    Label("//pathtools:path.go"),
    Label("//repo:BUILD.bazel"),
    Label("//repo:remote.go"),
    Label("//repo:repo.go"),
    Label("//resolve:BUILD.bazel"),
    Label("//resolve:config.go"),
    Label("//resolve:index.go"),
    Label("//rule:BUILD.bazel"),
    Label("//rule:directives.go"),
    Label("//rule:expr.go"),
    Label("//rule:merge.go"),
    Label("//rule:platform.go"),
    Label("//rule:platform_strings.go"),
    Label("//rule:rule.go"),
    Label("//rule:sort_labels.go"),
    Label("//rule:types.go"),
    Label("//rule:value.go"),
    Label("//testtools:BUILD.bazel"),
    Label("//testtools:config.go"),
    Label("//testtools:files.go"),
    Label("//tools:BUILD.bazel"),
    Label("//tools/override-generator:BUILD.bazel"),
    Label("//tools/override-generator:main.go"),
    Label("//tools/releaser:BUILD.bazel"),
    Label("//tools/releaser:main.go"),
    Label("//walk:BUILD.bazel"),
    Label("//walk:config.go"),
    Label("//walk:walk.go"),
]
