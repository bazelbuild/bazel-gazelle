visibility([
    "//tests/bzlmod/...",
])

# structs have certain built-in methods or attributes that should not
# be overwritten. Load these by calling `dir` on an empty struct.
# e.g. ["to_proto", "to_json"]
_STRUCT_PROTECTED_ATTRIBUTES = dir(struct())

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
            # Treat "gazelle:key    value" the same as "gazelle:key value".
            value = directive[len(prefix):].lstrip()

    return value

def with_replaced_or_new_fields(_struct, **replacements):
    """Provides a shallow copy of a structure with replacements and/or new fields

    Args:
        _struct: structure to shallow copy.
        **replacements: kwargs for fields to either replace or add to the new struct.

    Returns:
        The resulting updated structure.
    """

    new_struct_assignments = {
        key: getattr(_struct, key)
        for key in dir(_struct)
        if key not in _STRUCT_PROTECTED_ATTRIBUTES
    }

    # Overwrite existing fields and add new ones.
    for key, value in replacements.items():
        new_struct_assignments[key] = value

    return struct(**new_struct_assignments)
