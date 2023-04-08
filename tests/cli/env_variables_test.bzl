load("@bazel_skylib//rules:analysis_test.bzl", "analysis_test")
load("@bazel_skylib//lib:unittest.bzl", "analysistest", "asserts", "unittest")
load("//:def.bzl", "gazelle")

def _invalid_var_failure_test_impl(ctx):
    env = analysistest.begin(ctx)
    var_name = ctx.attr.var_name
    if var_name == None:
        unittest.fail(env, "var_name can't be null")
    asserts.expect_failure(env, "Invalid environmental variable name: '%s'" % var_name)
    return analysistest.end(env)

invalid_var_failure_test = analysistest.make(
    _invalid_var_failure_test_impl,
    expect_failure = True,
    attrs = {
        "var_name": attr.string(),
    },
)

def env_variables_test_suite():
    gazelle(
        name = "gazelle-valid-env-variables",
        env = {
            "SOME_VARIABLE": "1",
            "YET_ANOTHER_VARIABLE_0": "2",
            "_another_variable": "3",
        },
        tags = ["manual"],
    )

    analysis_test(
        name = "valid_env_variables_test",
        targets = [
            ":gazelle-valid-env-variables",
        ],
    )

    gazelle(
        name = "gazelle-numbers-in-var-name",
        env = {
            "0foo": "",
        },
        tags = ["manual"],
    )

    invalid_var_failure_test(
        name = "env_variable_name_cant_start_with_numbers_test",
        target_under_test = ":gazelle-numbers-in-var-name",
        var_name = "0foo",
    )

    gazelle(
        name = "gazelle-empty-var-name",
        env = {
            "": "",
        },
        tags = ["manual"],
    )

    invalid_var_failure_test(
        name = "env_variable_name_cant_be_empty",
        target_under_test = ":gazelle-empty-var-name",
        var_name = "",
    )
