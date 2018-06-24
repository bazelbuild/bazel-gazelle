def _check_file(test, path, gazelle, args):
    """Return shell commands for testing file 'f'."""

# We write information to stdout. It will show up in logs, so that the user
# knows what happened if the test fails.

def _impl(ctx):
  script = """
#! /usr/bin/env bash
"./{test}" "{path}" "${{PWD}}/{gazelle}" "{args}"
""".format(
    test=ctx.executable._test.path,
    path=ctx.attr.path,
    gazelle=ctx.executable._gazelle.short_path,
    args=" ".join(ctx.attr.args),
  )

  # Write the file, it is executed by 'bazel test'.
  ctx.actions.write(
      output = ctx.outputs.executable,
      content = script,
  )

  # To ensure the files needed by the script are available, we put them in
  # the runfiles.
  runfiles = ctx.runfiles(files = ctx.files.srcs + [
      ctx.executable._gazelle,
      ctx.executable._test,
  ])
  return [DefaultInfo(runfiles = runfiles)]

  """
  Gazelle Test rule

  Generate a test that creates a minimal working environment for Gazelle to
  operate in, execute Gazelle in that directory and compare the generated BUILD
  files to the snapshotted goldens that preexist under the name "BUILD.in"

  Args:
    name of the test. This also doubles as the directory name that the testing
      files are under.
  """

gazelle_test = rule(
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "_gazelle": attr.label(
            default = Label("//cmd/gazelle:gazelle_pure"),
            allow_files = True,
            executable = True,
            cfg = "host",
        ),
        "_test": attr.label(
            default = Label("//testing:test.sh"),
            allow_files = True,
            executable = True,
            cfg = "host",
        ),
        "path": attr.string(),
    },
    test = True,
    implementation = _impl,
)
