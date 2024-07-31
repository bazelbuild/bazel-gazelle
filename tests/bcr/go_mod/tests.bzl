load("@rules_license//rules:providers.bzl", "PackageInfo")
load("@rules_testing//lib:analysis_test.bzl", "analysis_test", "test_suite")
load("@rules_testing//lib:truth.bzl", "subjects")

def _test_package_info(name):
    analysis_test(
        name = name,
        impl = _test_package_info_impl,
        target = "@com_github_fmeum_dep_on_gazelle//:go_default_library",
        extra_target_under_test_aspects = [
            _package_info_aspect,
        ],
        provider_subject_factories = [_PackageInfoSubjectFactory],
    )

def _test_package_info_impl(env, target):
    env.expect.that_target(target).has_provider(PackageInfo)
    subject = env.expect.that_target(target).provider(PackageInfo)
    subject.package_name().equals("github.com/fmeum/dep_on_gazelle")
    subject.package_version().equals("1.0.0")
    subject.package_url().equals("https://github.com/fmeum/dep_on_gazelle")

def _package_info_aspect_impl(_, ctx):
    if hasattr(ctx.rule.attr, "applicable_licenses"):
        attr = ctx.rule.attr.applicable_licenses
    elif hasattr(ctx.rule.attr, "package_metadata"):
        attr = ctx.rule.attr.package_metadata
    if PackageInfo in attr[0]:
        return [attr[0][PackageInfo]]
    return []

_package_info_aspect = aspect(
    implementation = _package_info_aspect_impl,
    doc = "Forwards metadata annotations on the target via the PackageInfo provider.",
)

_PackageInfoSubjectFactory = struct(
    type = PackageInfo,
    name = "PackageInfo",
    factory = lambda actual, *, meta: subjects.struct(
        actual,
        meta = meta,
        attrs = {
            "package_name": subjects.str,
            "package_version": subjects.str,
            "package_url": subjects.str,
        },
    ),
)

def starlark_tests(name):
    test_suite(
        name = name,
        tests = [
            _test_package_info,
        ],
    )
