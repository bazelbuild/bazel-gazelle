load("//internal:go_repository.bzl", "go_repository")
load(":go_mod.bzl", "deps_from_go_mod", "sums_from_go_mod")
load(
    ":default_gazelle_overrides.bzl",
    "DEFAULT_BUILD_FILE_GENERATION_BY_PATH",
    "DEFAULT_DIRECTIVES_BY_PATH",
)
load(":semver.bzl", "semver")
load(
    ":utils.bzl",
    "drop_nones",
    "format_rule_call",
    "get_directive_value",
    "with_replaced_or_new_fields",
)

visibility("//")

_HIGHEST_VERSION_SENTINEL = semver.to_comparable("999999.999999.999999")

_FORBIDDEN_OVERRIDE_TAG = """\
Using the "go_deps.{tag_class}" tag in a non-root Bazel module is forbidden, \
but module "{module_name}" requests it.

If you need this override for a Bazel module that will be available in a public \
registry (such as the Bazel Central Registry), please file an issue at \
https://github.com/bazelbuild/bazel-gazelle/issues/new or submit a PR adding \
the required directives to the "default_gazelle_overrides.bzl" file at \
https://github.com/bazelbuild/bazel-gazelle/tree/master/internal/bzlmod/default_gazelle_overrides.bzl.
"""

def _fail_on_non_root_overrides(module, tag_class):
    if module.is_root:
        return

    if getattr(module.tags, tag_class):
        fail(_FORBIDDEN_OVERRIDE_TAG.format(
            tag_class = tag_class,
            module_name = module.name,
        ))

def _check_directive(directive):
    if directive.startswith("gazelle:") and " " in directive and not directive[len("gazelle:"):][0].isspace():
        return
    fail("Invalid Gazelle directive: \"{}\". Gazelle directives must be of the form \"gazelle:key value\".".format(directive))

def _get_build_file_generation(path, gazelle_overrides):
    override = gazelle_overrides.get(path)
    if override:
        return override.build_file_generation

    return DEFAULT_BUILD_FILE_GENERATION_BY_PATH.get(path, "auto")

def _get_directives(path, gazelle_overrides):
    override = gazelle_overrides.get(path)
    if override:
        return override.directives

    return DEFAULT_DIRECTIVES_BY_PATH.get(path, [])

def _get_patches(path, module_overrides):
    override = module_overrides.get(path)
    if override:
        return override.patches
    return []

def _get_patch_args(path, module_overrides):
    override = module_overrides.get(path)
    if override:
        return ["-p{}".format(override.patch_strip)]
    return []

def _repo_name(importpath):
    path_segments = importpath.split("/")
    segments = reversed(path_segments[0].split(".")) + path_segments[1:]
    candidate_name = "_".join(segments).replace("-", "_")
    return "".join([c.lower() if c.isalnum() else "_" for c in candidate_name.elems()])

def _is_dev_dependency(module_ctx, tag):
    if hasattr(tag, "_is_dev_dependency"):
        # Synthetic tags generated from go_deps.from_file have this "hidden" attribute.
        return tag._is_dev_dependency

    # This function is available in Bazel 6.2.0 and later. This is the same version that has
    # module_ctx.extension_metadata, so the return value of this function is not used if it is
    # not available.
    return module_ctx.is_dev_dependency(tag) if hasattr(module_ctx, "is_dev_dependency") else False

def _extension_metadata(module_ctx, *, root_module_direct_deps, root_module_direct_dev_deps):
    if not hasattr(module_ctx, "extension_metadata"):
        return None
    return module_ctx.extension_metadata(
        root_module_direct_deps = root_module_direct_deps,
        root_module_direct_dev_deps = root_module_direct_dev_deps,
    )

def _go_repository_config_impl(ctx):
    repos = []
    for name, importpath in sorted(ctx.attr.importpaths.items()):
        repos.append(format_rule_call(
            "go_repository",
            name = name,
            importpath = importpath,
            module_name = ctx.attr.module_names.get(name),
            build_naming_convention = ctx.attr.build_naming_conventions.get(name),
        ))

    ctx.file("WORKSPACE", "\n".join(repos))
    ctx.file("BUILD.bazel", "exports_files(['WORKSPACE'])")

_go_repository_config = repository_rule(
    implementation = _go_repository_config_impl,
    attrs = {
        "importpaths": attr.string_dict(mandatory = True),
        "module_names": attr.string_dict(mandatory = True),
        "build_naming_conventions": attr.string_dict(mandatory = True),
    },
)

def _noop(_):
    pass

def _go_deps_impl(module_ctx):
    module_resolutions = {}
    sums = {}
    replace_map = {}
    bazel_deps = {}

    gazelle_overrides = {}
    module_overrides = {}

    root_versions = {}
    root_fixups = []
    root_module_direct_deps = {}
    root_module_direct_dev_deps = {}

    if module_ctx.modules[0].name == "gazelle":
        root_module_direct_deps["bazel_gazelle_go_repository_config"] = None

    outdated_direct_dep_printer = print
    for module in module_ctx.modules:
        # Parse the go_deps.config tag of the root module only.
        for mod_config in module.tags.config:
            if not module.is_root:
                continue
            check_direct_deps = mod_config.check_direct_dependencies
            if check_direct_deps == "off":
                outdated_direct_dep_printer = _noop
            elif check_direct_deps == "warning":
                outdated_direct_dep_printer = print
            elif check_direct_deps == "error":
                outdated_direct_dep_printer = fail

        _fail_on_non_root_overrides(module, "gazelle_override")
        for gazelle_override_tag in module.tags.gazelle_override:
            if gazelle_override_tag.path in gazelle_overrides:
                fail("Multiple overrides defined for Go module path \"{}\" in module \"{}\".".format(gazelle_override_tag.path, module.name))
            for directive in gazelle_override_tag.directives:
                _check_directive(directive)

            gazelle_overrides[gazelle_override_tag.path] = struct(
                directives = gazelle_override_tag.directives,
                build_file_generation = gazelle_override_tag.build_file_generation,
            )

        _fail_on_non_root_overrides(module, "module_override")
        for module_override_tag in module.tags.module_override:
            if module_override_tag.path in module_overrides:
                fail("Multiple overrides defined for Go module path \"{}\" in module \"{}\".".format(module_override_tag.path, module.name))
            module_overrides[module_override_tag.path] = struct(
                patches = module_override_tag.patches,
                patch_strip = module_override_tag.patch_strip,
            )

        if len(module.tags.from_file) > 1:
            fail(
                "Multiple \"go_deps.from_file\" tags defined in module \"{}\": {}".format(
                    module.name,
                    ", ".join([str(tag.go_mod) for tag in module.tags.from_file]),
                ),
            )
        additional_module_tags = []
        for from_file_tag in module.tags.from_file:
            module_path, module_tags_from_go_mod, go_mod_replace_map = deps_from_go_mod(module_ctx, from_file_tag.go_mod)
            is_dev_dependency = _is_dev_dependency(module_ctx, from_file_tag)
            additional_module_tags += [
                with_replaced_or_new_fields(tag, _is_dev_dependency = is_dev_dependency)
                for tag in module_tags_from_go_mod
            ]

            if module.is_root:
                replace_map.update(go_mod_replace_map)
            else:
                # Register this Bazel module as providing the specified Go module. It participates
                # in version resolution using its registry version, which uses a relaxed variant of
                # semver that can however still be compared to strict semvers.
                # An empty version string signals an override, which is assumed to be newer than any
                # other version.
                raw_version = _canonicalize_raw_version(module.version)
                version = semver.to_comparable(raw_version, relaxed = True) if raw_version else _HIGHEST_VERSION_SENTINEL
                if module_path not in bazel_deps or version > bazel_deps[module_path].version:
                    bazel_deps[module_path] = struct(
                        module_name = module.name,
                        repo_name = "@" + from_file_tag.go_mod.workspace_name,
                        version = version,
                        raw_version = raw_version,
                    )

            # Load all sums from transitively resolved `go.sum` files that have modules.
            if len(module_tags_from_go_mod) > 0:
                for entry, new_sum in sums_from_go_mod(module_ctx, from_file_tag.go_mod).items():
                    _safe_insert_sum(sums, entry, new_sum)

        # Load sums from manually specified modules separately.
        for module_tag in module.tags.module:
            if module_tag.build_naming_convention:
                fail("""The "build_naming_convention" attribute is no longer supported for "go_deps.module" tags. Use a "gazelle:go_naming_convention" directive via the "gazelle_override" tag's "directives" attribute instead.""")
            if module_tag.build_file_proto_mode:
                fail("""The "build_file_proto_mode" attribute is no longer supported for "go_deps.module" tags. Use a "gazelle:proto" directive via the "gazelle_override" tag's "directives" attribute instead.""")
            sum_version = _canonicalize_raw_version(module_tag.version)
            _safe_insert_sum(sums, (module_tag.path, sum_version), module_tag.sum)

        # Parse the go_dep.module tags of all transitive dependencies and apply
        # Minimum Version Selection to resolve importpaths to Go module versions
        # and sums.
        #
        # Note: This applies Minimum Version Selection on the resolved
        # dependency graphs of all transitive Bazel module dependencies, which
        # is not what `go mod` does. But since this algorithm ends up using only
        # Go module versions that have been explicitly declared somewhere in the
        # full graph, we can assume that at that place all its required
        # transitive dependencies have also been declared - we may end up
        # resolving them to higher versions, but only compatible ones.
        paths = {}
        for module_tag in module.tags.module + additional_module_tags:
            if module_tag.path in paths:
                fail("Duplicate Go module path \"{}\" in module \"{}\".".format(module_tag.path, module.name))
            if module_tag.path in bazel_deps:
                continue
            paths[module_tag.path] = None
            raw_version = _canonicalize_raw_version(module_tag.version)

            # For modules imported from a go.sum, we know which ones are direct
            # dependencies and can thus only report implicit version upgrades
            # for direct dependencies. For manually specified go_deps.module
            # tags, we always report version upgrades unless users override with
            # the "indirect" attribute.
            if module.is_root and not module_tag.indirect:
                root_versions[module_tag.path] = raw_version
                if _is_dev_dependency(module_ctx, module_tag):
                    root_module_direct_dev_deps[_repo_name(module_tag.path)] = None
                else:
                    root_module_direct_deps[_repo_name(module_tag.path)] = None

            version = semver.to_comparable(raw_version)
            if module_tag.path not in module_resolutions or version > module_resolutions[module_tag.path].version:
                module_resolutions[module_tag.path] = struct(
                    repo_name = _repo_name(module_tag.path),
                    version = version,
                    raw_version = raw_version,
                )

    unmatched_gazelle_overrides = []
    for path in gazelle_overrides.keys():
        if path not in module_resolutions:
            unmatched_gazelle_overrides.append(path)
    if unmatched_gazelle_overrides:
        fail("Some gazelle_overrides did not target a Go module with a matching path: {}"
            .format(", ".join(unmatched_gazelle_overrides)))

    # All `replace` directives are applied after version resolution.
    # We can simply do this by checking the replace paths' existence
    # in the module resolutions and swapping out the entry.
    for path, replace in replace_map.items():
        if path in module_resolutions:
            new_version = semver.to_comparable(replace.version)
            module_resolutions[path] = with_replaced_or_new_fields(
                module_resolutions[path],
                replace = replace.to_path,
                version = new_version,
                raw_version = replace.version,
            )
            if path in root_versions:
                if replace != replace.to_path:
                    # If the root module replaces a Go module with a completely different one, do
                    # not ever report an implicit version upgrade.
                    root_versions.pop(path)
                else:
                    root_versions[path] = replace.version

    for path, bazel_dep in bazel_deps.items():
        # We can't apply overrides to Bazel dependencies and thus fall back to using the Go module.
        if path in gazelle_overrides or path in module_overrides or path in replace_map:
            continue

        # Only use the Bazel module if it is at least as high as the required Go module version.
        if path in module_resolutions and bazel_dep.version < module_resolutions[path].version:
            outdated_direct_dep_printer(
                "Go module \"{path}\" is provided by Bazel module \"{bazel_module}\" in version {bazel_dep_version}, but requested at higher version {go_version} via Go requirements. Consider adding or updating an appropriate \"bazel_dep\" to ensure that the Bazel module is used to provide the Go module.".format(
                    path = path,
                    bazel_module = bazel_dep.module_name,
                    bazel_dep_version = bazel_dep.raw_version,
                    go_version = module_resolutions[path].raw_version,
                ),
            )
            continue

        # TODO: We should update root_versions if the bazel_dep is a direct dependency of the root
        #   module. However, we currently don't have a way to determine that.
        module_resolutions[path] = bazel_dep

    for path, root_version in root_versions.items():
        if semver.to_comparable(root_version) < module_resolutions[path].version:
            outdated_direct_dep_printer(
                "For Go module \"{path}\", the root module requires module version v{root_version}, but got v{resolved_version} in the resolved dependency graph.".format(
                    path = path,
                    root_version = root_version,
                    resolved_version = module_resolutions[path].raw_version,
                ),
            )

    for path, module in module_resolutions.items():
        if hasattr(module, "module_name"):
            # Do not create a go_repository for a Go module provided by a bazel_dep.
            root_module_direct_deps.pop(_repo_name(path), default = None)
            root_module_direct_dev_deps.pop(_repo_name(path), default = None)
            continue

        go_repository(
            name = module.repo_name,
            importpath = path,
            sum = _get_sum_from_module(path, module, sums),
            replace = getattr(module, "replace", None),
            version = "v" + module.raw_version,
            build_directives = _get_directives(path, gazelle_overrides),
            build_file_generation = _get_build_file_generation(path, gazelle_overrides),
            patches = _get_patches(path, module_overrides),
            patch_args = _get_patch_args(path, module_overrides),
        )

    # Create a synthetic WORKSPACE file that lists all Go repositories created
    # above and contains all the information required by Gazelle's -repo_config
    # to generate BUILD files for external Go modules. This skips the need to
    # run generate_repo_config. Only "importpath" and "build_naming_convention"
    # are relevant.
    _go_repository_config(
        name = "bazel_gazelle_go_repository_config",
        importpaths = {
            module.repo_name: path
            for path, module in module_resolutions.items()
        },
        module_names = {
            info.repo_name: info.module_name
            for path, info in bazel_deps.items()
        },
        build_naming_conventions = drop_nones({
            module.repo_name: get_directive_value(
                _get_directives(path, gazelle_overrides),
                "go_naming_convention",
            )
            for path, module in module_resolutions.items()
        }),
    )

    return _extension_metadata(
        module_ctx,
        root_module_direct_deps = root_module_direct_deps.keys(),
        # If a Go module appears as both a dev and a non-dev dependency, it has to be imported as a
        # non-dev dependency.
        root_module_direct_dev_deps = {
            repo_name: None
            for repo_name in root_module_direct_dev_deps.keys()
            if repo_name not in root_module_direct_deps
        }.keys(),
    )

def _get_sum_from_module(path, module, sums):
    entry = (path, module.raw_version)
    if hasattr(module, "replace"):
        entry = (module.replace, module.raw_version)

    if entry not in sums:
        fail("No sum for {}@{} found".format(path, module.raw_version))

    return sums[entry]

def _safe_insert_sum(sums, entry, new_sum):
    if entry in sums and new_sum != sums[entry]:
        fail("Multiple mismatching sums for {}@{} found.".format(entry[0], entry[1]))
    sums[entry] = new_sum

def _canonicalize_raw_version(raw_version):
    if raw_version.startswith("v"):
        return raw_version[1:]
    return raw_version

_config_tag = tag_class(
    attrs = {
        "check_direct_dependencies": attr.string(
            values = ["off", "warning", "error"],
        ),
    },
)

_from_file_tag = tag_class(
    attrs = {
        "go_mod": attr.label(mandatory = True),
    },
)

_module_tag = tag_class(
    attrs = {
        "path": attr.string(mandatory = True),
        "version": attr.string(mandatory = True),
        "sum": attr.string(),
        "indirect": attr.bool(
            doc = """Whether this Go module is an indirect dependency.""",
            default = False,
        ),
        "build_naming_convention": attr.string(doc = """Removed, do not use""", default = ""),
        "build_file_proto_mode": attr.string(doc = """Removed, do not use""", default = ""),
    },
)

_gazelle_override_tag = tag_class(
    attrs = {
        "path": attr.string(
            doc = """The Go module path for the repository to be overridden.

            This module path must be defined by other tags in this
            extension within this Bazel module.""",
            mandatory = True,
        ),
        "build_file_generation": attr.string(
            default = "auto",
            doc = """One of `"auto"` (default), `"on"`, `"off"`.

            Whether Gazelle should generate build files for the Go module. In
            `"auto"` mode, Gazelle will run if there is no build file in the Go
            module's root directory.""",
            values = [
                "auto",
                "off",
                "on",
            ],
        ),
        "directives": attr.string_list(
            doc = """Gazelle configuration directives to use for this Go module's external repository.

            Each directive uses the same format as those that Gazelle
            accepts as comments in Bazel source files, with the
            directive name followed by optional arguments separated by
            whitespace.""",
        ),
    },
    doc = "Override Gazelle's behavior on a given Go module defined by other tags in this extension.",
)

_module_override_tag = tag_class(
    attrs = {
        "path": attr.string(
            doc = """The Go module path for the repository to be overridden.

            This module path must be defined by other tags in this
            extension within this Bazel module.""",
            mandatory = True,
        ),
        "patches": attr.label_list(
            doc = "A list of patches to apply to the repository *after* gazelle runs.",
        ),
        "patch_strip": attr.int(
            default = 0,
            doc = "The number of leading path segments to be stripped from the file name in the patches.",
        ),
    },
    doc = "Apply patches to a given Go module defined by other tags in this extension.",
)

go_deps = module_extension(
    _go_deps_impl,
    tag_classes = {
        "config": _config_tag,
        "from_file": _from_file_tag,
        "gazelle_override": _gazelle_override_tag,
        "module": _module_tag,
        "module_override": _module_override_tag,
    },
)
