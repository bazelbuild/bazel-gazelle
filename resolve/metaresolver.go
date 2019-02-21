package resolve

import (
	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// MetaResolver provides a rule.Resolver for any rule.Rule.
type MetaResolver struct {
	// builtins provides a map of the language kinds to their resolver.
	builtins map[string]Resolver

	// replKinds provides a list of replacements used by File.Pkg.
	replKinds map[string][]config.ReplacementKind
}

func NewMetaResolver() *MetaResolver {
	return &MetaResolver{
		builtins:  make(map[string]Resolver),
		replKinds: make(map[string][]config.ReplacementKind),
	}
}

// AddBuiltin registers a builtin kind with its info.
func (mr *MetaResolver) AddBuiltin(kindName string, resolver Resolver) {
	mr.builtins[kindName] = resolver
}

// MappedKind records the fact that the given mapping was applied while
// processing the given file.
func (mr *MetaResolver) MappedKind(f *rule.File, kind config.ReplacementKind) {
	mr.replKinds[f.Pkg] = append(mr.replKinds[f.Pkg], kind)
}

// Resolver returns a resolver for the given rule and file, and a bool
// indicating whether one was found.  If f is nil, mapped kinds are disregarded.
func (mr MetaResolver) Resolver(r *rule.Rule, f *rule.File) Resolver {
	// If f is provided, check the replacements used while processing that package.
	// If the rule is a kind that was mapped, return the resolver for the kind it was mapped from.
	if f != nil {
		for _, replKind := range mr.replKinds[f.Pkg] {
			if replKind.KindName == r.Kind() {
				return mr.builtins[replKind.FromKind]
			}
		}
	}
	return mr.builtins[r.Kind()]
}
