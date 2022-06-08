load("@bazel_skylib//lib:unittest.bzl", "asserts", "unittest")
load("//internal/bzlmod:semver.bzl", "semver")

_SORTED_TEST_VERSIONS = [
    "1.0.0-0.3.7",
    "1.0.0-alpha",
    "1.0.0-alpha+001",
    "1.0.0-alpha.1",
    "1.0.0-alpha.beta",
    "1.0.0-beta+exp.sha.5114f85",
    "1.0.0-beta",
    "1.0.0-beta.2",
    "1.0.0-beta.11",
    "1.0.0-rc.1",
    "1.0.0-x.7.z.92",
    "1.0.0-x-y-z.–",
    "1.0.0+21AF26D3—-117B344092BD",
    "1.0.0+20130313144700",
    "1.0.0",
    "2.0.0",
    "2.1.0",
    "2.1.1",
]

_SCRAMBLED_TEST_VERSIONS = [
    "2.1.1",
    "2.1.0",
    "2.0.0",
    "1.0.0+21AF26D3—-117B344092BD",
    "1.0.0+20130313144700",
    "1.0.0",
    "1.0.0-x.7.z.92",
    "1.0.0-x-y-z.–",
    "1.0.0-rc.1",
    "1.0.0-beta.11",
    "1.0.0-beta.2",
    "1.0.0-beta+exp.sha.5114f85",
    "1.0.0-beta",
    "1.0.0-alpha.beta",
    "1.0.0-alpha.1",
    "1.0.0-alpha",
    "1.0.0-alpha+001",
    "1.0.0-0.3.7",
]

def _semver_test_impl(ctx):
    env = unittest.begin(ctx)
    asserts.equals(
        env,
        _SORTED_TEST_VERSIONS,
        sorted(_SCRAMBLED_TEST_VERSIONS, key = semver.to_comparable),
    )
    return unittest.end(env)

semver_test = unittest.make(_semver_test_impl)

def semver_test_suite(name):
    unittest.suite(
        name,
        semver_test,
    )
