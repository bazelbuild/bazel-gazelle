package label

import (
	"fmt"
	"path"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/pathtools"
)

type Pattern struct {
	Repo          string
	Pkg           string
	Recursive     bool
	IsExplicitAll bool
	SpecificName  string
}

// NoPattern is the zero value of Pattern. It is not a valid pattern and may be
// returned when an error occurs.
var NoPattern = Pattern{}

// ParsePattern parses a pattern which may match some Labels.
// See https://docs.bazel.build/versions/main/guide.html#target-patterns
// If non-absolute patterns are passed, they are assumed to be relative to the repo root.
//
// Only a single pattern may be matched, no compound of complex query patterns
// (i.e. there is no support for parsing `foo/... + bar/...`),
// and negative patterns are not supported
// (i.e. there is no support for parsing -//foo/...).
func ParsePattern(s string) (Pattern, error) {
	if !strings.HasPrefix(s, "//") && !strings.HasPrefix(s, "@") {
		s = "//" + s
	}
	if strings.HasSuffix(s, ":*") {
		s = s[:len(s)-1] + "all"
	} else if strings.HasSuffix(s, ":all-targets") {
		s = s[:len(s)-11] + "all"
	}

	label, err := Parse(s)
	if err != nil {
		return NoPattern, err
	}
	repo := label.Repo
	recursive := false
	isExplicitAll := false
	specificName := ""

	if strings.HasSuffix(label.Pkg, "...") {
		recursive = true
		if label.Name == "..." {
			label.Name = ""
		}

		if label.Pkg == "..." {
			label.Pkg = ""
		} else {
			label.Pkg = label.Pkg[0 : len(label.Pkg)-4]
		}
	}
	pkg := label.Pkg

	if label.Name == "all" {
		isExplicitAll = true
	} else {
		specificName = label.Name
	}
	return Pattern{
		Repo:          repo,
		Pkg:           pkg,
		Recursive:     recursive,
		IsExplicitAll: isExplicitAll,
		SpecificName:  specificName,
	}, nil
}

// Matches returns whether a label matches a pattern.
// See https://bazel.build/docs/build#specifying-build-targets
// Relative patterns are assumed to be relative to the repo root.
// This comparison is purely lexical - no analysis is performed such as whether
// a label refers to a non-default output of a rule.
func (p *Pattern) Matches(l Label) bool {
	if p.Repo != l.Repo {
		return false
	}
	if p.Pkg == l.Pkg {
		return p.IsExplicitAll || p.SpecificName == l.Name || p.Recursive
	}
	return p.Recursive && pathtools.HasPrefix(l.Pkg, p.Pkg)
}

func (p Pattern) String() string {
	var repo string
	if p.Repo != "" && p.Repo != "@" {
		repo = fmt.Sprintf("@%s", p.Repo)
	} else {
		repo = p.Repo
	}
	var name string
	if p.IsExplicitAll {
		name = ":all"
	} else if p.SpecificName != "" && path.Base(p.Pkg) != p.SpecificName {
		name = fmt.Sprintf(":%s", p.SpecificName)
	}
	var dotDotDot string
	if p.Recursive {
		if p.Pkg == "" {
			dotDotDot = "..."
		} else {
			dotDotDot = "/..."
		}
	}
	return fmt.Sprintf("%s//%s%s%s", repo, p.Pkg, dotDotDot, name)
}
