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

package config

// MergableAttrs is the set of attribute names for each kind of rule that
// may be merged. When an attribute is mergeable, a generated value may
// replace or augment an existing value. If an attribute is not mergeable,
// existing values are preserved. Generated non-mergeable attributes may
// still be added to a rule if there is no corresponding existing attribute.
type MergeableAttrs map[string]map[string]bool
