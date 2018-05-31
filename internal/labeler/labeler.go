/* Copyright 2017 The Bazel Authors. All rights reserved.

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

package labeler

import (
	"github.com/bazelbuild/bazel-gazelle/internal/config"
	"github.com/bazelbuild/bazel-gazelle/internal/label"
	"github.com/bazelbuild/bazel-gazelle/internal/pathtools"
)

// Labeler generates Bazel labels for rules, based on their locations
// within the repository.
//
// TODO(jayconrod): delete this type (and this package). When we have
// multiple rule generators for different languages, it doesn't make sense
// to group this logic in a single place.
type Labeler struct {
	c *config.Config
}

func NewLabeler(c *config.Config) *Labeler {
	return &Labeler{c}
}

func (l *Labeler) LibraryLabel(rel string) label.Label {
	return label.Label{Pkg: rel, Name: config.DefaultLibName}
}

func (l *Labeler) TestLabel(rel string) label.Label {
	return label.Label{Pkg: rel, Name: config.DefaultTestName}
}

func (l *Labeler) BinaryLabel(rel string) label.Label {
	name := pathtools.RelBaseName(rel, l.c.GoPrefix, l.c.RepoRoot)
	return label.Label{Pkg: rel, Name: name}
}

func (l *Labeler) ProtoLabel(rel, name string) label.Label {
	return label.Label{Pkg: rel, Name: name + "_proto"}
}

func (l *Labeler) GoProtoLabel(rel, name string) label.Label {
	return label.Label{Pkg: rel, Name: name + "_go_proto"}
}
