load("//internal:go_repository.bzl", "go_repository")
load(":go_mod.bzl", "deps_from_go_mod")
load(":directives.bzl", "DEFAULT_DIRECTIVES_BY_PATH")
load(":semver.bzl", "semver")
load(":utils.bzl", "drop_nones", "format_rule_call", "get_directive_value")

visibility("//")

# These Go modules are imported as Bazel modules via bazel_dep, not as
# go_repository.
IGNORED_MODULE_PATHS = [
    "github.com/bazelbuild/bazel-gazelle",
    "github.com/bazelbuild/rules_go",
]

_FORBIDDEN_OVERRIDE_TAG = """\
Using the "go_deps.{tag_class}" tag in a non-root Bazel module is forbidden, \
but module "{module_name}" requests it.
"""

_FORBIDDEN_OVERRIDE_ATTRIBUTE = """\
Using the "{attribute}" attribute in a "go_deps.{tag_class}" tag is forbidden \
in non-root Bazel modules, but module "{module_name}" requests it.
"""

_DIRECTIVES_CALL_TO_ACTION = """\

If you need this override for a Bazel module that will be available in a public \
registry (such as the Bazel Central Registry), please file an issue at \
https://github.com/bazelbuild/bazel-gazelle/issues/new or submit a PR adding \
the required directives to the "directives.bzl" file at \
https://github.com/bazelbuild/bazel-gazelle/tree/master/internal/bzlmod/directives.bzl.
"""

def _report_forbidden_override(module, tag_class, attribute = None):
    if attribute:
        message = _FORBIDDEN_OVERRIDE_ATTRIBUTE.format(
            attribute = attribute,
            tag_class = tag_class,
            module_name = module.name,
        )
    else:
        message = _FORBIDDEN_OVERRIDE_TAG.format(
            tag_class = tag_class,
            module_name = module.name,
        )

    return message + _DIRECTIVES_CALL_TO_ACTION

def _fail_on_non_root_overrides(module, tag_class, attribute = None):
    # TODO: Gazelle and the "rules_go" module depend on each other circularly.
    #  Tolerate overrides in the latter module until we can update it to no
    #  longer need them.
    if module.is_root or module.name == "rules_go":
        return

    tags = getattr(module.tags, tag_class)
    for tag in tags:
        if attribute:
            if getattr(tag, attribute):
                fail(_report_forbidden_override(module, tag_class, attribute))
        else:
            fail(_report_forbidden_override(module, tag_class))

def _check_directive(directive):
    if directive.startswith("gazelle:") and " " in directive:
        return
    fail("Invalid Gazelle directive: \"{}\". Gazelle directives must be of the form \"gazelle:key value\".".format(directive))

def _synthesize_gazelle_override(module, gazelle_overrides):
    """Translate deprecated override attributes to directives for a transition period."""
    directives = []

    build_naming_convention = getattr(module, "build_naming_convention", "")
    if build_naming_convention:
        directive = "gazelle:go_naming_convention " + build_naming_convention
        directives.append(directive)

    build_file_proto_mode = getattr(module, "build_file_proto_mode", "")
    if build_file_proto_mode:
        directive = "gazelle:proto " + build_file_proto_mode
        directives.append(directive)

    if directives:
        gazelle_overrides[module.path] = struct(
            directives = directives,
        )

def _get_directives(path, gazelle_overrides):
    override = gazelle_overrides.get(path)
    if override:
        return override.directives

    return DEFAULT_DIRECTIVES_BY_PATH.get(path, [])

def _repo_name(importpath):
    path_segments = importpath.split("/")
    segments = reversed(path_segments[0].split(".")) + path_segments[1:]
    candidate_name = "_".join(segments).replace("-", "_")
    return "".join([c.lower() if c.isalnum() else "_" for c in candidate_name.elems()])

def _go_repository_config_impl(ctx):
    repos = []
    for name, importpath in sorted(ctx.attr.importpaths.items()):
        repos.append(format_rule_call(
            "go_repository",
            name = name,
            importpath = importpath,
            build_naming_convention = ctx.attr.build_naming_conventions.get(name),
        ))

    ctx.file("WORKSPACE", "\n".join(repos))
    ctx.file("BUILD.bazel")

_go_repository_config = repository_rule(
    implementation = _go_repository_config_impl,
    attrs = {
        "importpaths": attr.string_dict(mandatory = True),
        "build_naming_conventions": attr.string_dict(mandatory = True),
    },
)

def _noop(_):
    pass

def _go_deps_impl(module_ctx):
    module_resolutions = {}
    gazelle_overrides = {}
    root_versions = {}

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

        for override_tag in module.tags.gazelle_override:
            if override_tag.path in gazelle_overrides:
                fail("Multiple overrides defined for Go module path \"{}\" in module \"{}\".".format(override_tag.path, module.name))
            for directive in override_tag.directives:
                _check_directive(directive)

            gazelle_overrides[override_tag.path] = struct(
                directives = override_tag.directives,
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
            additional_module_tags += deps_from_go_mod(module_ctx, from_file_tag.go_mod)

        _fail_on_non_root_overrides(module, "module", "build_naming_convention")
        _fail_on_non_root_overrides(module, "module", "build_file_proto_mode")

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
            if module_tag.path in IGNORED_MODULE_PATHS:
                continue
            paths[module_tag.path] = None
            raw_version = module_tag.version
            if raw_version.startswith("v"):
                raw_version = raw_version[1:]

            # Note: While we still have overrides in rules_go, those will take precedence over the
            #  ones defined in the root module.
            _synthesize_gazelle_override(module_tag, gazelle_overrides)

            # For modules imported from a go.sum, we know which ones are direct
            # dependencies and can thus only report implicit version upgrades
            # for direct dependencies. For manually specified go_deps.module
            # tags, we always report version upgrades.
            if module.is_root and getattr(module_tag, "direct", True):
                root_versions[module_tag.path] = raw_version
            version = semver.to_comparable(raw_version)
            if module_tag.path not in module_resolutions or version > module_resolutions[module_tag.path].version:
                module_resolutions[module_tag.path] = struct(
                    module = module.name,
                    repo_name = _repo_name(module_tag.path),
                    version = version,
                    raw_version = raw_version,
                    sum = module_tag.sum,
                )

    unmatched_gazelle_overrides = []
    for path in gazelle_overrides.keys():
        if path not in module_resolutions:
            unmatched_gazelle_overrides.append(path)
    if unmatched_gazelle_overrides:
        fail("Some gazelle_overrides did not target a Go module with a matching path: {}"
            .format(", ".join(unmatched_gazelle_overrides)))

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
        go_repository(
            name = module.repo_name,
            importpath = path,
            sum = module.sum,
            version = "v" + module.raw_version,
            build_directives = _get_directives(path, gazelle_overrides),
        )

    # Create a synthetic WORKSPACE file that lists all Go repositories created
    # above and contains all the information required by Gazelle's -repo_config
    # to generate BUILD files for external Go modules. This skips the need to
    # run generate_repo_config. Only 'importpath' and 'build_naming_convention'
    # are relevant.
    _go_repository_config(
        name = "bazel_gazelle_go_repository_config",
        importpaths = {
            module.repo_name: path
            for path, module in module_resolutions.items()
        },
        build_naming_conventions = drop_nones({
            module.repo_name: get_directive_value(
                _get_directives(path, gazelle_overrides),
                "go_naming_convention",
            )
            for path, module in module_resolutions.items()
        }),
    )

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
        "build_naming_convention": attr.string(
            doc = """The library naming convention to use when
            resolving dependencies against this Go module's external
            repository.

            Deprecated: Use the "gazelle:build_file_names" directive
            via gazelle_override tag's "directive" attribute
            instead.""",
            default = "",
            values = [
                "go_default_library",
                "import",
                "import_alias",
            ],
        ),
        "build_file_proto_mode": attr.string(
            doc = """The mode to use when generating rules for
            Protocol Buffers files for this Go module's external
            repository.

            Deprecated: Use the "gazelle:proto" directive via
            gazelle_override tag's "build_file_proto_mode" attribute
            instead.""",
            default = "",
            values = [
                "default",
                "disable",
                "disable_global",
                "legacy",
                "package",
            ],
        ),
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

go_deps = module_extension(
    _go_deps_impl,
    tag_classes = {
        "config": _config_tag,
        "from_file": _from_file_tag,
        "gazelle_override": _gazelle_override_tag,
        "module": _module_tag,
    },
)
