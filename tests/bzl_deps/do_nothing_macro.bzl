"""Macro used with bzl_test

For more information, please see `bzl_test.bzl`.
"""

load("//:def.bzl", "DEFAULT_LANGUAGES")

def macro_with_doc(name):
    """This macro does nothing.

    Args:
        name: A `string` value.
    """
    if name == None:
        return None
    return DEFAULT_LANGUAGES
