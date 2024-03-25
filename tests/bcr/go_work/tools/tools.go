//go:build tools

// Follows the best practice from https://github.com/golang/go/issues/25922#issuecomment-1038394599
// to keep Go tools referenced in go.mod in a way that doesn't have `go mod tidy` remove them.

package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/99designs/gqlgen/graphql/introspection"
	_ "github.com/bazelbuild/buildtools/buildifier"
)
