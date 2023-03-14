def _is_version_suffix(s):
    if not s or s[0] != "v":
        return False
    return s[1:].isdigit()

def _lib_name_from_import_path(import_path):
    """Starlark implementation of libNameFromImportPath in package.go."""
    i = import_path.rfind("/")
    if i < 0:
        # TODO: Should we replace '.' with '_' also in this case?
        #  libNameFromImportPath doesn't do that, but that seems wrong.
        return import_path
    name = import_path[i + 1:]
    if _is_version_suffix(name):
        unversioned_import_path = import_path[:i]
        i = unversioned_import_path.rfind("/")
        if i >= 0:
            name = unversioned_import_path[i + 1:]
    return name.replace(".", "_")

def go_helper(import_path, module_path_to_repo, canonicalize):
    split_pos = len(import_path)
    for i in range(import_path.count("/") + 1):
        module_path = import_path[:split_pos]
        module_repo = module_path_to_repo.get(module_path, None)
        if module_repo:
            package_path = import_path[split_pos:].removeprefix("/")
            return canonicalize("@{}//{}:{}".format(module_repo, package_path, _lib_name_from_import_path(import_path)))
        split_pos = module_path.rfind("/", 0, split_pos)

    fail("""

No Go module in go.mod provides package '{import_path}'.
Add the missing dependency via:

    go get {import_path}

""".format(import_path = import_path))
