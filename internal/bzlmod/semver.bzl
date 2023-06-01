visibility([
    "//tests/bzlmod/...",
])

# Compares lower than any non-numeric identifier.
_COMPARES_LOWEST_SENTINEL = ""

# Compares higher than any valid non-numeric identifier (containing only [A-Za-z0-9-]).
_COMPARES_HIGHEST_SENTINEL = "{"

def _identifier_to_comparable(ident, *, numeric_only):
    if not ident:
        fail("Identifiers in semantic version strings must not be empty")
    if ident.isdigit():
        if ident[0] == "0" and ident != "0":
            fail("Numeric identifiers in semantic version strings must not include leading zeroes")

        # 11.4.1:
        # "Identifiers consisting of only digits are compared numerically."
        # 11.4.3:
        # "Numeric identifiers always have lower precedence than non-numeric identifiers."
        return (_COMPARES_LOWEST_SENTINEL, int(ident))
    elif numeric_only:
        fail("Expected a numeric identifier, got: " + ident)
    else:
        # 11.4.2:
        # "Identifiers with letters or hyphens are compared lexically in ASCII sort order."
        return (ident,)

def _semver_to_comparable(v, *, relaxed = False):
    """
    Parses a string representation of a semver version into an opaque comparable object.

    Args:
        v: The string representation of the version.
        relaxed: If true, the release version string is allowed to have an arbitrary number of
            dot-separated components, each of which is allowed to contain the same set of characters
            as a pre-release segment. This is the version string format used by Bazel modules.
    """

    # Strip build metadata as it is not relevant for comparisons.
    v, _, _ = v.partition("+")

    release_str, _, prerelease_str = v.partition("-")
    if prerelease_str:
        # 11.4.4:
        # "A larger set of pre-release fields has a higher precedence than a smaller set, if all of the preceding
        #  identifiers are equal."
        prerelease = [_identifier_to_comparable(ident, numeric_only = False) for ident in prerelease_str.split(".")]
    else:
        # 11.3:
        # "When major, minor, and patch are equal, a pre-release version has lower precedence than a normal version."
        prerelease = [(_COMPARES_HIGHEST_SENTINEL,)]

    release = release_str.split(".")
    if not relaxed and len(release) != 3:
        fail("Semantic version strings must have exactly three dot-separated components, got: " + v)

    return (
        tuple([_identifier_to_comparable(s, numeric_only = not relaxed) for s in release]),
        tuple(prerelease),
    )

semver = struct(
    to_comparable = _semver_to_comparable,
)
