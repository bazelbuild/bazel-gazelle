/* Copyright 2022 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package golang

import (
	"bufio"
	"bytes"
	"fmt"
	"go/build/constraint"
	"os"
	"strings"
)

// buildTags represents the build tags specified in a file.
type buildTags struct {
	// expr represents the parsed constraint expression
	// that can be used to evaluate a file against a set
	// of tags.
	expr constraint.Expr
	// rawTags represents the concrete tags that make up expr.
	rawTags []string
}

// newBuildTags will return a new buildTags structure with any
// ignored tags filtered out from the provided constraints.
func newBuildTags(x constraint.Expr) (*buildTags, error) {
	filtered, err := filterTags(x, func(tag string) bool {
		return !isIgnoredTag(tag)
	})
	if err != nil {
		return nil, err
	}

	rawTags, err := collectTags(x)
	if err != nil {
		return nil, err
	}

	return &buildTags{
		expr:    filtered,
		rawTags: rawTags,
	}, nil
}

func (b *buildTags) tags() []string {
	if b == nil {
		return nil
	}

	return b.rawTags
}

func (b *buildTags) eval(ok func(string) bool) bool {
	if b == nil || b.expr == nil {
		return true
	}

	return b.expr.Eval(ok)
}

func (b *buildTags) empty() bool {
	if b == nil {
		return true
	}

	return len(b.rawTags) == 0
}

// filterTags will traverse the provided constraint.Expr, recursively, and call
// the user provided ok func on concrete constraint.TagExpr structures. If the provided
// func returns true, the tag in question is kept, otherwise it is filtered out.
func filterTags(expr constraint.Expr, ok func(string) bool) (constraint.Expr, error) {
	if expr == nil {
		return nil, nil
	}

	switch x := expr.(type) {
	case *constraint.TagExpr:
		if ok(x.Tag) {
			return &constraint.TagExpr{Tag: x.Tag}, nil
		}

	case *constraint.NotExpr:
		filtered, err := filterTags(x.X, ok)
		if err != nil {
			return nil, err
		}

		if filtered != nil {
			return &constraint.NotExpr{X: filtered}, nil
		}

	case *constraint.AndExpr:
		a, err := filterTags(x.X, ok)
		if err != nil {
			return nil, err
		}

		b, err := filterTags(x.Y, ok)
		if err != nil {
			return nil, err
		}

		// An AND constraint requires two operands.
		// If either is no longer present due to recursive
		// filtering, then return the non-nil value.
		if a != nil && b != nil {
			return &constraint.AndExpr{
				X: a,
				Y: b,
			}, nil

		} else if a != nil {
			return a, nil

		} else if b != nil {
			return b, nil
		}

	case *constraint.OrExpr:
		a, err := filterTags(x.X, ok)
		if err != nil {
			return nil, err
		}

		b, err := filterTags(x.Y, ok)
		if err != nil {
			return nil, err
		}

		// An OR constraint requires two operands.
		// If either is no longer present due to recursive
		// filtering, then return the non-nil value.
		if a != nil && b != nil {
			return &constraint.OrExpr{
				X: a,
				Y: b,
			}, nil

		} else if a != nil {
			return a, nil

		} else if b != nil {
			return b, nil
		}

	default:
		return nil, fmt.Errorf("unknown constraint type: %T", x)
	}

	return nil, nil
}

func collectTags(expr constraint.Expr) ([]string, error) {
	var tags []string
	_, err := filterTags(expr, func(tag string) bool {
		tags = append(tags, tag)
		return true
	})
	if err != nil {
		return nil, err
	}

	return tags, err
}

// cgoTagsAndOpts contains compile or link options which should only be applied
// if the given set of build tags are satisfied. These options have already
// been tokenized using the same algorithm that "go build" uses, then joined
// with OptSeparator.
type cgoTagsAndOpts struct {
	*buildTags
	opts string
}

func (c *cgoTagsAndOpts) tags() []string {
	if c == nil {
		return nil
	}

	return c.buildTags.tags()
}

func (c *cgoTagsAndOpts) eval(ok func(string) bool) bool {
	if c == nil {
		return true
	}

	return c.buildTags.eval(ok)
}

// readTags reads and extracts build tags from the block of comments
// and blank lines at the start of a file which is separated from the
// rest of the file by a blank line. Each string in the returned slice
// is the trimmed text of a line after a "+build" prefix.
// Based on go/build.Context.shouldBuild.
func readTags(path string) (*buildTags, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := readComments(f)
	if err != nil {
		return nil, err
	}

	content, goBuild, _, err := parseFileHeader(content)
	if err != nil {
		return nil, err
	}

	if goBuild != nil {
		x, err := constraint.Parse(string(goBuild))
		if err != nil {
			return nil, err
		}

		return newBuildTags(x)
	}

	var fullConstraint constraint.Expr
	// Search and parse +build tags
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if !constraint.IsPlusBuild(line) {
			continue
		}

		x, err := constraint.Parse(line)
		if err != nil {
			return nil, err
		}

		if fullConstraint != nil {
			fullConstraint = &constraint.AndExpr{
				X: fullConstraint,
				Y: x,
			}
		} else {
			fullConstraint = x
		}
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	if fullConstraint == nil {
		return nil, nil
	}

	return newBuildTags(fullConstraint)
}

// matchAuto interprets text as either a +build or //go:build expression (whichever works).
// Forked from go/build.Context.matchAuto
func matchAuto(tokens []string) (*buildTags, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	text := strings.Join(tokens, " ")
	if strings.ContainsAny(text, "&|()") {
		text = "//go:build " + text
	} else {
		text = "// +build " + text
	}

	x, err := constraint.Parse(text)
	if err != nil {
		return nil, err
	}

	return newBuildTags(x)
}

// isIgnoredTag returns whether the tag is "cgo" or is a release tag.
// Release tags match the pattern "go[0-9]\.[0-9]+".
// Gazelle won't consider whether an ignored tag is satisfied when evaluating
// build constraints for a file.
func isIgnoredTag(tag string) bool {
	if tag == "cgo" || tag == "race" || tag == "msan" {
		return true
	}
	if len(tag) < 5 || !strings.HasPrefix(tag, "go") {
		return false
	}
	if tag[2] < '0' || tag[2] > '9' || tag[3] != '.' {
		return false
	}
	for _, c := range tag[4:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
