/* Copyright 2018 The Bazel Authors. All rights reserved.

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

package proto

import "path/filepath"

// protoPackage contains metadata for a set of .proto files that have the
// same package name. This translates to a proto_library rule.
type protoPackage struct {
	name    string
	files   map[string]FileInfo
	imports map[string]bool
	options map[string]string
}

func newProtoPackage(name string) *protoPackage {
	return &protoPackage{
		name:    name,
		files:   map[string]FileInfo{},
		imports: map[string]bool{},
		options: map[string]string{},
	}
}

func (p *protoPackage) addFile(info FileInfo) {
	p.files[info.Name] = info
	for _, imp := range info.Imports {
		p.imports[imp] = true
	}
	for _, opt := range info.Options {
		p.options[opt.Key] = opt.Value
	}
}

func (p *protoPackage) addGenFile(dir, name string) {
	p.files[name] = FileInfo{
		Name: name,
		Path: filepath.Join(dir, filepath.FromSlash(name)),
	}
}
