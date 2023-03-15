visibility("private")

def drop_nones(dict):
    """Drop entries with None values from a dictionary.

    Args:
        dict: A dictionary.

    Returns:
        A new dictionary with the same keys as dict, but with entries with
        None values removed.
    """
    return {k: v for k, v in dict.items() if v != None}

def format_rule_call(_kind, **attrs):
    """Format a rule call.

    Args:
        _kind: The kind of the rule.
        attrs: The attributes of the rule. The attributes are sorted by name,
            except for the "name" attribute, which is always first. None
            values are ignored.

    Returns:
        A string representing the rule call.
    """

    lines = [_kind + "("]
    for attr, value in sorted(attrs.items(), key = _attrs_sort_key):
        if value == None:
            continue
        lines.append("    {} = {},".format(attr, repr(value)))
    lines.append(")")

    return "\n".join(lines)

def _attrs_sort_key(entry):
    """Sort key for attributes.

    Attributes are sorted by name, except for the "name" attribute, which is
    always first.

    Args:
        entry: A tuple of (attribute name, attribute value).

    Returns:
        An opaque sort key.
    """
    attr, _ = entry
    if attr == "name":
        # Sort "name" first.
        return ""
    return attr

def get_directive_value(directives, key):
    """Get the value of a directive.

    Args:
        directives: A list of directives.
        key: The key of the directive.

    Returns:
        The value of the last directive, or None if the directive is not
        present.
    """
    prefix = "gazelle:" + key + " "

    value = None
    for directive in directives:
        if directive.startswith(prefix):
            value = directive[len(prefix):]

    return value

def buildozer_cmd(*args, name = "all"):
    return struct(
        args = args,
        name = name,
    )

def format_module_file_fixup(commands):
    """Format a fixup suggestion for a MODULE.bazel file.

    Args:
        commands: A list of return values of buildozer_cmd.

    Returns:
        A buildozer script performing the fixup.
    """
    script_lines = [_format_buildozer_command(cmd) for cmd in commands]
    return """
To migrate automatically, paste the following lines into a file and run it with \
'buildozer -f <file>', using at least buildozer 6.0.0:

""" + "\n".join(script_lines) + "\n\n"

def _format_buildozer_command(cmd):
    return "{args}|//MODULE.bazel:{name}".format(
        args = " ".join([arg.replace(" ", "\\ ") for arg in cmd.args]),
        name = cmd.name,
    )
