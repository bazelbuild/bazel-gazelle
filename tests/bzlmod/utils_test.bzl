load("@bazel_skylib//lib:unittest.bzl", "asserts", "unittest")
load("//internal/bzlmod:utils.bzl", "with_replaced_or_new_fields", "repo_name")

_BEFORE_STRUCT = struct(
    direct = True,
    path = "github.com/bazelbuild/buildtools",
    version = "v0.0.0-20220531122519-a43aed7014c8",
)

_EXPECT_REPLACED_STRUCT = struct(
    direct = True,
    path = "github.com/bazelbuild/buildtools",
    replace = "path/to/add/replace",
    version = "v1.2.2",
)

def _with_replaced_or_new_fields_test_impl(ctx):
    env = unittest.begin(ctx)
    asserts.equals(env, _EXPECT_REPLACED_STRUCT, with_replaced_or_new_fields(
        _BEFORE_STRUCT,
        replace = "path/to/add/replace",
        version = "v1.2.2",
    ))
    return unittest.end(env)

with_replaced_or_new_fields_test = unittest.make(_with_replaced_or_new_fields_test_impl)

def _repo_name_test_impl(ctx):
    env = unittest.begin(ctx)
    asserts.equals(env, "com_github_shurcoo_l_githubv4", repo_name("github.com/shurcooL/githubv4"))
    asserts.equals(env, "com_github_shurcool_githubv4", repo_name("github.com/shurcool/githubv4"))
    asserts.equals(env, "com_github__d_a_t_a__d_o_g_go_sqlmock", repo_name("github.com/DATA-DOG/go-sqlmock"))
    return unittest.end(env)

repo_name_test = unittest.make(_repo_name_test_impl)

def utils_test_suite(name):
    unittest.suite(
        name,
        with_replaced_or_new_fields_test,
        repo_name_test,
    )
