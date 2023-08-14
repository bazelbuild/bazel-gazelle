visibility([
    "//tests/bzlmod/...",
])

def deps_from_go_mod(module_ctx, go_mod_label):
    """Loads the entries from a go.mod file.

    Args:
        module_ctx: a https://bazel.build/rules/lib/module_ctx object passed
            from the MODULE.bazel call.
        go_mod_label: a Label for a `go.mod` file.

    Returns:
        a tuple (Go module path, deps, replace map), where deps is a list of structs representing
        `require` statements from the go.mod file.
    """
    _check_go_mod_name(go_mod_label.name)

    go_mod_path = module_ctx.path(go_mod_label)
    go_mod_content = module_ctx.read(go_mod_path)
    go_mod = parse_go_mod(go_mod_content, go_mod_path)

    if go_mod.go[0] != 1 or go_mod.go[1] < 17:
        # go.mod files only include entries for all transitive dependencies as
        # of Go 1.17.
        fail("go_deps.from_file requires a go.mod file generated by Go 1.17 or later. Fix {} with 'go mod tidy -go=1.17'.".format(go_mod_label))

    deps = []
    for require in go_mod.require:
        deps.append(struct(
            path = require.path,
            version = require.version,
            indirect = require.indirect,
        ))

    return go_mod.module, deps, go_mod.replace_map

def parse_go_mod(content, path):
    # See https://go.dev/ref/mod#go-mod-file.

    # Valid directive values understood by this parser never contain tabs or
    # carriage returns, so we can simplify the parsing below by canonicalizing
    # whitespace upfront.
    content = content.replace("\t", " ").replace("\r", " ")

    state = {
        "module": None,
        "go": None,
        "require": [],
        "replace": {},
    }

    current_directive = None
    for line_no, line in enumerate(content.splitlines(), 1):
        tokens, comment = _tokenize_line(line, path, line_no)
        if not tokens:
            continue

        if not current_directive:
            if tokens[0] not in ["module", "go", "require", "replace", "exclude", "retract", "toolchain"]:
                fail("{}:{}: unexpected token '{}' at start of line".format(path, line_no, tokens[0]))
            if len(tokens) == 1:
                fail("{}:{}: expected another token after '{}'".format(path, line_no, tokens[0]))

            # The 'go' directive only has a single-line form and is thus parsed
            # here rather than in _parse_directive.
            if tokens[0] == "go":
                if len(tokens) == 1:
                    fail("{}:{}: expected another token after 'go'".format(path, line_no))
                if state["go"] != None:
                    fail("{}:{}: unexpected second 'go' directive".format(path, line_no))
                state["go"] = tokens[1]
                if len(tokens) > 2:
                    fail("{}:{}: unexpected token '{}' after '{}'".format(path, line_no, tokens[2], tokens[1]))

            if tokens[1] == "(":
                current_directive = tokens[0]
                if len(tokens) > 2:
                    fail("{}:{}: unexpected token '{}' after '('".format(path, line_no, tokens[2]))
                continue

            _parse_directive(state, tokens[0], tokens[1:], comment, path, line_no)

        elif tokens[0] == ")":
            current_directive = None
            if len(tokens) > 1:
                fail("{}:{}: unexpected token '{}' after ')'".format(path, line_no, tokens[1]))
            continue

        else:
            _parse_directive(state, current_directive, tokens, comment, path, line_no)

    module = state["module"]
    if not module:
        fail("Expected a module directive in go.mod file")

    go = state["go"]
    if not go:
        # "As of the Go 1.17 release, if the go directive is missing, go 1.16 is assumed."
        go = "1.16"

    # The go directive can contain patch and pre-release versions, but we omit them.
    major, minor = go.split(".")[:2]

    return struct(
        module = module,
        go = (int(major), int(minor)),
        require = tuple(state["require"]),
        replace_map = state["replace"],
    )

def _parse_directive(state, directive, tokens, comment, path, line_no):
    if directive == "module":
        if state["module"] != None:
            fail("{}:{}: unexpected second 'module' directive".format(path, line_no))
        if len(tokens) > 1:
            fail("{}:{}: unexpected token '{}' after '{}'".format(path, line_no, tokens[1]))
        state["module"] = tokens[0]
    elif directive == "require":
        if len(tokens) != 2:
            fail("{}:{}: expected module path and version in 'require' directive".format(path, line_no))
        state["require"].append(struct(
            path = tokens[0],
            version = tokens[1],
            indirect = comment == "indirect",
        ))
    elif directive == "replace":
        # A replace directive might use a local file path beginning with ./ or ../
        # These are not supported with gazelle~go_deps.
        if len(tokens) == 3 and tokens[2][0] == ".":
            fail("{}:{}: local file path not supported in replace directive: '{}'".format(path, line_no, tokens[2]))

        if len(tokens) != 4 or tokens[1] != "=>":
            fail("{}:{}: replace directive must follow pattern: 'replace from_path => to_path version' ".format(path, line_no))
        from_path = tokens[0]
        state["replace"][from_path] = struct(
            to_path = tokens[2],
            version = _canonicalize_raw_version(tokens[3]),
        )

    # TODO: Handle exclude.

def _tokenize_line(line, path, line_no):
    tokens = []
    r = line
    for _ in range(len(line)):
        r = r.strip()
        if not r:
            break

        if r[0] == "`":
            end = r.find("`", 1)
            if end == -1:
                fail("{}:{}: unterminated raw string".format(path, line_no))

            tokens.append(r[1:end])
            r = r[end + 1:]

        elif r[0] == "\"":
            value = ""
            escaped = False
            found_end = False
            for pos in range(1, len(r)):
                c = r[pos]

                if escaped:
                    value += c
                    escaped = False
                    continue

                if c == "\\":
                    escaped = True
                    continue

                if c == "\"":
                    found_end = True
                    break

                value += c

            if not found_end:
                fail("{}:{}: unterminated interpreted string".format(path, line_no))

            tokens.append(value)
            r = r[pos + 1:]

        elif r.startswith("//"):
            # A comment always ends the current line
            return tokens, r[len("//"):].strip()

        else:
            token, _, r = r.partition(" ")
            tokens.append(token)

    return tokens, None

def sums_from_go_mod(module_ctx, go_mod_label):
    """Loads the entries from a go.sum file given a go.mod Label.

    Args:
        module_ctx: a https://bazel.build/rules/lib/module_ctx object
            passed from the MODULE.bazel call.
        go_mod_label: a Label for a `go.mod` file. This label is used
            to find the associated `go.sum` file.

    Returns:
        A Dict[(string, string) -> (string)] is retruned where each entry
        is defined by a Go Module's sum:
            (path, version) -> (sum)
    """
    _check_go_mod_name(go_mod_label.name)

    # We go through a Label so that the module extension is restarted if go.sum
    # changes. We have to use a canonical label as we may not have visibility
    # into the module that provides the go.sum.
    go_sum_label = Label("@@{}//{}:{}".format(
        go_mod_label.workspace_name,
        go_mod_label.package,
        "go.sum",
    ))
    go_sum_content = module_ctx.read(go_sum_label)
    return parse_go_sum(go_sum_content)

def parse_go_sum(content):
    hashes = {}
    for line in content.splitlines():
        path, version, sum = line.split(" ")
        version = _canonicalize_raw_version(version)
        if not version.endswith("/go.mod"):
            hashes[(path, version)] = sum
    return hashes

def _check_go_mod_name(name):
    if name != "go.mod":
        fail("go_deps.from_file requires a 'go.mod' file, not '{}'".format(name))

def _canonicalize_raw_version(raw_version):
    if raw_version.startswith("v"):
        return raw_version[1:]
    return raw_version
