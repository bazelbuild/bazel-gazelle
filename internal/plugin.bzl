load("@io_bazel_rules_go//go/private:providers.bzl", "GoLibrary")

GazellePlugin = provider("Gazelle plugin provider", fields = [
    # Constructor is the name of the method inside the importpath provided by
    # library that constructs an object that implements the language.Language
    # interface.
    "constructor",
    # Library is a GoLibrary that can be passed into the
    "library",
])

def _gazelle_plugin_impl(ctx):
    return [GazellePlugin(
        library = ctx.attr.library,
        constructor = ctx.attr.constructor,
    )]

gazelle_plugin = rule(
    implementation = _gazelle_plugin_impl,
    attrs = {
        "constructor": attr.string(),
        "library": attr.label(providers = [GoLibrary]),
    },
)
