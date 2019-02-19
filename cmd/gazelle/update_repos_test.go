/* Copyright 2019 The Bazel Authors. All rights reserved.

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

package main

import (
	"testing"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/testtools"
)

func testConfig(t *testing.T, args ...string) (*config.Config, []config.Configurer) {
	// Add a -repo_root argument if none is present. Without this,
	// config.CommonConfigurer will try to auto-detect a WORKSPACE file,
	// which will fail.
	args = append(args, "-repo_root=.")
	cexts := []config.Configurer{&config.CommonConfigurer{}, &updateReposConfigurer{}}
	c := testtools.NewTestConfig(t, cexts, nil, args)
	return c, cexts
}

func TestCommandLine(t *testing.T) {
	c, _ := testConfig(
		t,
		"-from_file=Gopkg.lock",
		"-build_file_names=BUILD",
		"-build_external=external",
		"-build_file_generation=on",
		"-build_tags=foo,bar",
		"-build_file_proto_mode=default",
		"-build_extra_args=-exclude=vendor",
	)
	uc := getUpdateReposConfig(c)
	if uc.lockFilename != "Gopkg.lock" {
		t.Errorf(`got from_file %q; want "Gopkg.lock"`, uc.lockFilename)
	}
	if uc.buildFileNamesAttr != "BUILD" {
		t.Errorf(`got build_file_name %q; want "BUILD"`, uc.buildFileNamesAttr)
	}
	if uc.buildExternalAttr != "external" {
		t.Errorf(`got build_external %q; want "external"`, uc.buildExternalAttr)
	}
	if uc.buildFileGenerationAttr != "on" {
		t.Errorf(`got build_file_generation %q; want "on"`, uc.buildFileGenerationAttr)
	}
	if uc.buildTagsAttr != "foo,bar" {
		t.Errorf(`got build_tags %q; want "foo,bar"`, uc.buildTagsAttr)
	}
	if uc.buildFileProtoModeAttr != "default" {
		t.Errorf(`got build_file_proto_mode %q; want "default"`, uc.buildFileProtoModeAttr)
	}
	if uc.buildExtraArgsAttr != "-exclude=vendor" {
		t.Errorf(`got build_file_proto_mode %q; want "-exclude=vendor"`, uc.buildExtraArgsAttr)
	}
}
