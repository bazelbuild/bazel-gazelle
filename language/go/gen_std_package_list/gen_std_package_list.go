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

// gen_std_package_list reads a text file containing a list of packages
// (one per line) and generates a .go file containing a set of package
// names. The text file is generated by an SDK repository rule. The
// set of package names is used by Gazelle to determine whether an
// import path is in the standard library.
package main

import (
	"bytes"
	"log"
	"os"
	"text/template"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) != 3 {
		log.Fatalf("usage: %s packages.txt out.go", os.Args[0])
	}

	packagesTxtPath := os.Args[1]
	genGoPath := os.Args[2]

	packagesTxt, err := os.ReadFile(packagesTxtPath)
	if err != nil {
		log.Fatal(err)
	}
	var newline []byte
	if bytes.HasSuffix(packagesTxt, []byte("\r\n")) {
		newline = []byte("\r\n")
	} else {
		newline = []byte("\n")
	}
	packagesTxt = bytes.TrimSuffix(packagesTxt, newline)
	packageList := bytes.Split(packagesTxt, newline)

	tmpl := template.Must(template.New("std_package_list").Parse(`
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

// Generated by gen_std_package_list.go
// DO NOT EDIT

package golang

var stdPackages = map[string]bool{
{{range . -}}
{{printf "\t%q" .}}: true,
{{end -}}
}
`))
	f, err := os.Create(genGoPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	err = tmpl.Execute(f, packageList)
	if err != nil {
		log.Fatal(err)
	}
}
