def gazelle_dependencies():
  _maybe(native.git_repository,
      name = "bazel_skylib",
      remote = "https://github.com/bazelbuild/bazel-skylib",
      commit = "f3dd8fd95a7d078cb10fd7fb475b22c3cdbcb307", # 0.2.0 as of 2017-12-04
  )

def _maybe(repo_rule, name, **kwargs):
  if name not in native.existing_rules():
    repo_rule(name=name, **kwargs)
