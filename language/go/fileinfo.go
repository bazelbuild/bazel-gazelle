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

package golang

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/internal/version"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

// fileInfo holds information used to decide how to build a file. This
// information comes from the file's name, from package and import declarations
// (in .go files), and from +build and cgo comments.
type fileInfo struct {
	path string
	name string

	// ext is the type of file, based on extension.
	ext ext

	// packageName is the Go package name of a .go file, without the
	// "_test" suffix if it was present. It is empty for non-Go files.
	packageName string

	// hasMainFunction is true when packageName is "main" and a main function was
	// found in the file
	hasMainFunction bool

	// isTest is true if the file stem (the part before the extension)
	// ends with "_test.go". This is never true for non-Go files.
	isTest bool

	// isExternalTest is true when the file isTest and the original package
	// name ends with "_test"
	isExternalTest bool

	// imports is a list of packages imported by a file. It does not include
	// "C" or anything from the standard library.
	imports []string

	// embeds is a list of //go:embed patterns and their positions.
	embeds []fileEmbed

	// isCgo is true for .go files that import "C".
	isCgo bool

	// goos and goarch contain the OS and architecture suffixes in the filename,
	// if they were present.
	goos, goarch string

	// tags is a list of build tag lines. Each entry is the trimmed text of
	// a line after a "+build" prefix.
	tags *buildTags

	// cppopts, copts, cxxopts and clinkopts contain flags that are part
	// of CPPFLAGS, CFLAGS, CXXFLAGS, and LDFLAGS directives in cgo comments.
	cppopts, copts, cxxopts, clinkopts []*cgoTagsAndOpts

	// hasServices indicates whether a .proto file has service definitions.
	hasServices bool
}

// fileEmbed represents an individual go:embed pattern.
// A go:embed directive may contain multiple patterns. A pattern may match
// multiple files.
type fileEmbed struct {
	path string
	pos  token.Position
}

// optSeparator is a special character inserted between options that appeared
// together in a #cgo directive. This allows options to be split, modified,
// and escaped by other packages.
//
// It's important to keep options grouped together in the same string. For
// example, if we have "-framework IOKit" together in a #cgo directive,
// "-framework" shouldn't be treated as a separate string for the purposes of
// sorting and de-duplicating.
const optSeparator = "\x1D"

// ext indicates how a file should be treated, based on extension.
type ext int

const (
	// unknownExt is applied files that aren't buildable with Go.
	unknownExt ext = iota

	// goExt is applied to .go files.
	goExt

	// cExt is applied to C and C++ files.
	cExt

	// hExt is applied to header files. If cgo code is present, these may be
	// C or C++ headers. If not, they are treated as Go assembly headers.
	hExt

	// sExt is applied to Go assembly files, ending with .s.
	sExt

	// csExt is applied to other assembly files, ending with .S. These are built
	// with the C compiler if cgo code is present.
	csExt

	// protoExt is applied to .proto files.
	protoExt
)

// fileNameInfo returns information that can be inferred from the name of
// a file. It does not read data from the file.
func fileNameInfo(path_ string) fileInfo {
	name := filepath.Base(path_)
	var ext ext
	switch path.Ext(name) {
	case ".go":
		ext = goExt
	case ".c", ".cc", ".cpp", ".cxx", ".m", ".mm":
		ext = cExt
	case ".h", ".hh", ".hpp", ".hxx":
		ext = hExt
	case ".s":
		ext = sExt
	case ".S":
		ext = csExt
	case ".proto":
		ext = protoExt
	default:
		ext = unknownExt
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		ext = unknownExt
	}

	// Determine test, goos, and goarch. This is intended to match the logic
	// in goodOSArchFile in go/build.
	var isTest bool
	var goos, goarch string
	l := strings.Split(name[:len(name)-len(path.Ext(name))], "_")
	if len(l) >= 2 && l[len(l)-1] == "test" {
		isTest = ext == goExt
		l = l[:len(l)-1]
	}
	switch {
	case len(l) >= 3 && rule.KnownOSSet[l[len(l)-2]] && rule.KnownArchSet[l[len(l)-1]]:
		goos = l[len(l)-2]
		goarch = l[len(l)-1]
	case len(l) >= 2 && rule.KnownOSSet[l[len(l)-1]]:
		goos = l[len(l)-1]
	case len(l) >= 2 && rule.KnownArchSet[l[len(l)-1]]:
		goarch = l[len(l)-1]
	}

	return fileInfo{
		path:   path_,
		name:   name,
		ext:    ext,
		isTest: isTest,
		goos:   goos,
		goarch: goarch,
	}
}

// otherFileInfo returns information about a non-.go file. It will parse
// part of the file to determine build tags. If the file can't be read, an
// error will be logged, and partial information will be returned.
func otherFileInfo(path string) fileInfo {
	info := fileNameInfo(path)
	if info.ext == unknownExt {
		return info
	}

	tags, err := readTags(info.path)
	if err != nil {
		log.Printf("%s: error reading file: %v", info.path, err)
		return info
	}
	info.tags = tags
	return info
}

// goFileInfo returns information about a .go file. It will parse part of the
// file to determine the package name, imports, and build constraints.
// If the file can't be read, an error will be logged, and partial information
// will be returned.
// This function is intended to match go/build.Context.Import.
// TODD(#53): extract canonical import path
func goFileInfo(path, srcdir string) fileInfo {
	info := fileNameInfo(path)
	fset := token.NewFileSet()
	pf, err := parser.ParseFile(fset, info.path, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		log.Printf("%s: error reading go file: %v", info.path, err)
		return info
	}

	info.packageName = pf.Name.Name
	if info.isTest && strings.HasSuffix(info.packageName, "_test") {
		info.packageName = info.packageName[:len(info.packageName)-len("_test")]
		info.isExternalTest = true
	}

	importsEmbed := false
	for _, decl := range pf.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, dspec := range d.Specs {
			spec, ok := dspec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			quoted := spec.Path.Value
			path, err := strconv.Unquote(quoted)
			if err != nil {
				log.Printf("%s: error reading go file: %v", info.path, err)
				continue
			}

			if path == "C" {
				if info.isTest {
					log.Printf("%s: warning: use of cgo in test not supported", info.path)
				}
				info.isCgo = true
				cg := spec.Doc
				if cg == nil && len(d.Specs) == 1 {
					cg = d.Doc
				}
				if cg != nil {
					if err := saveCgo(&info, srcdir, cg); err != nil {
						log.Printf("%s: error reading go file: %v", info.path, err)
					}
				}
				continue
			}
			if path == "embed" {
				importsEmbed = true
			}
			info.imports = append(info.imports, path)
		}
	}

	tags, err := readTags(info.path)
	if err != nil {
		log.Printf("%s: error reading go file: %v", info.path, err)
		return info
	}
	info.tags = tags

	if importsEmbed || info.packageName == "main" {
		pf, err = parser.ParseFile(fset, info.path, nil, parser.ParseComments)
		if err != nil {
			log.Printf("%s: error reading go file: %v", info.path, err)
			return info
		}
		for _, cg := range pf.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, "//go:embed") {
					continue
				}
				args := c.Text[len("//go:embed"):]
				p := c.Pos()
				for len(args) > 0 && (args[0] == ' ' || args[0] == '\t') {
					args = args[1:]
					p++
				}
				args = strings.TrimSpace(args) // trim right side
				pos := fset.Position(p)
				embeds, err := parseGoEmbed(args, pos)
				if err != nil {
					log.Printf("%v: parsing //go:embed directive: %v", pos, err)
					continue
				}
				info.embeds = append(info.embeds, embeds...)
			}
		}
		for _, decl := range pf.Decls {
			if fdecl, ok := decl.(*ast.FuncDecl); ok {
				if fdecl.Name.Name == "main" {
					info.hasMainFunction = true
					break
				}
			}
		}
	}

	return info
}

// saveCgo extracts CFLAGS, CPPFLAGS, CXXFLAGS, and LDFLAGS directives
// from a comment above a "C" import. This is intended to match logic in
// go/build.Context.saveCgo.
func saveCgo(info *fileInfo, srcdir string, cg *ast.CommentGroup) error {
	text := cg.Text()
	for _, line := range strings.Split(text, "\n") {
		orig := line

		// Line is
		//	#cgo [GOOS/GOARCH...] LDFLAGS: stuff
		//
		line = strings.TrimSpace(line)
		if len(line) < 5 || line[:4] != "#cgo" || (line[4] != ' ' && line[4] != '\t') {
			continue
		}

		// Split at colon.
		line, argstr, ok := strings.Cut(strings.TrimSpace(line[4:]), ":")
		if !ok {
			return fmt.Errorf("%s: invalid #cgo line: %s", info.path, orig)
		}

		// Parse tags and verb.
		f := strings.Fields(line)
		if len(f) < 1 {
			return fmt.Errorf("%s: invalid #cgo line: %s", info.path, orig)
		}

		cond, verb := f[:len(f)-1], f[len(f)-1]
		tags, err := matchAuto(cond)
		if err != nil {
			return err
		}

		// Parse options.
		opts, err := splitQuoted(argstr)
		if err != nil {
			return fmt.Errorf("%s: invalid #cgo line: %s", info.path, orig)
		}

		for i, opt := range opts {
			if opt, ok = expandSrcDir(opt, srcdir); !ok {
				return fmt.Errorf("%s: malformed #cgo argument: %s", info.path, orig)
			}
			opts[i] = opt
		}
		joinedStr := strings.Join(opts, optSeparator)

		// Add tags to appropriate list.
		switch verb {
		case "CPPFLAGS":
			info.cppopts = append(info.cppopts, &cgoTagsAndOpts{tags, joinedStr})
		case "CFLAGS":
			info.copts = append(info.copts, &cgoTagsAndOpts{tags, joinedStr})
		case "CXXFLAGS":
			info.cxxopts = append(info.cxxopts, &cgoTagsAndOpts{tags, joinedStr})
		case "FFLAGS":
			// TODO: Add support
			return fmt.Errorf("%s: unsupported #cgo verb: %s", verb, info.path)
		case "LDFLAGS":
			info.clinkopts = append(info.clinkopts, &cgoTagsAndOpts{tags, joinedStr})
		case "pkg-config":
			return fmt.Errorf("%s: pkg-config not supported: %s", info.path, orig)
		default:
			return fmt.Errorf("%s: invalid #cgo verb: %s", info.path, orig)
		}
	}
	return nil
}

// splitQuoted splits the string s around each instance of one or more consecutive
// white space characters while taking into account quotes and escaping, and
// returns an array of substrings of s or an empty list if s contains only white space.
// Single quotes and double quotes are recognized to prevent splitting within the
// quoted region, and are removed from the resulting substrings. If a quote in s
// isn't closed err will be set and r will have the unclosed argument as the
// last element. The backslash is used for escaping.
//
// For example, the following string:
//
//	a b:"c d" 'e''f'  "g\""
//
// Would be parsed as:
//
//	[]string{"a", "b:c d", "ef", `g"`}
//
// Copied from go/build.splitQuoted
func splitQuoted(s string) (r []string, err error) {
	var args []string
	arg := make([]rune, len(s))
	escaped := false
	quoted := false
	quote := '\x00'
	i := 0
	for _, rune := range s {
		switch {
		case escaped:
			escaped = false
		case rune == '\\':
			escaped = true
			continue
		case quote != '\x00':
			if rune == quote {
				quote = '\x00'
				continue
			}
		case rune == '"' || rune == '\'':
			quoted = true
			quote = rune
			continue
		case unicode.IsSpace(rune):
			if quoted || i > 0 {
				quoted = false
				args = append(args, string(arg[:i]))
				i = 0
			}
			continue
		}
		arg[i] = rune
		i++
	}
	if quoted || i > 0 {
		args = append(args, string(arg[:i]))
	}
	if quote != 0 {
		err = errors.New("unclosed quote")
	} else if escaped {
		err = errors.New("unfinished escaping")
	}
	return args, err
}

// expandSrcDir expands any occurrence of ${SRCDIR}, making sure
// the result is safe for the shell.
//
// Copied from go/build.expandSrcDir
func expandSrcDir(str string, srcdir string) (string, bool) {
	// "\" delimited paths cause safeCgoName to fail
	// so convert native paths with a different delimiter
	// to "/" before starting (eg: on windows).
	srcdir = filepath.ToSlash(srcdir)
	if srcdir == "" {
		srcdir = "."
	}

	// Spaces are tolerated in ${SRCDIR}, but not anywhere else.
	chunks := strings.Split(str, "${SRCDIR}")
	if len(chunks) < 2 {
		return str, safeCgoName(str, false)
	}
	ok := true
	for _, chunk := range chunks {
		ok = ok && (chunk == "" || safeCgoName(chunk, false))
	}
	ok = ok && (srcdir == "" || safeCgoName(srcdir, true))
	res := strings.Join(chunks, srcdir)
	return res, ok && res != ""
}

// NOTE: $ is not safe for the shell, but it is allowed here because of linker options like -Wl,$ORIGIN.
// We never pass these arguments to a shell (just to programs we construct argv for), so this should be okay.
// See golang.org/issue/6038.
// The @ is for OS X. See golang.org/issue/13720.
// The % is for Jenkins. See golang.org/issue/16959.
// The ~ is for Bzlmod as it is used as a separator in repository names. It is special in shells (which we don't pass
// it to) and on Windows (where it can still be part of legitimate short paths).
const (
	safeString = "+-.,/0123456789=ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz:$@%~"
	safeSpaces = " "
)

var safeBytes = []byte(safeSpaces + safeString)

// Copied from go/build.safeCgoName
func safeCgoName(s string, spaces bool) bool {
	if s == "" {
		return false
	}
	safe := safeBytes
	if !spaces {
		safe = safe[len(safeSpaces):]
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < utf8.RuneSelf && bytes.IndexByte(safe, c) < 0 {
			return false
		}
	}
	return true
}

func isOSArchSpecific(info fileInfo, cgoTags *cgoTagsAndOpts) (osSpecific, archSpecific bool) {
	if info.goos != "" {
		osSpecific = true
	}
	if info.goarch != "" {
		archSpecific = true
	}

	checkTags := func(tags []string) {
		for _, tag := range tags {
			_, osOk := rule.KnownOSSet[tag]
			if osOk || tag == "unix" {
				osSpecific = true
			}
			_, archOk := rule.KnownArchSet[tag]
			if archOk {
				archSpecific = true
			}
		}
	}
	checkTags(info.tags.tags())
	if osSpecific && archSpecific {
		return
	}

	checkTags(cgoTags.tags())
	return
}

// matchesOS checks if a value is equal to either an OS value or to any of its
// aliases.
func matchesOS(os, value string) bool {
	if os == value {
		return true
	}
	if value == "unix" {
		return rule.UnixOS[os]
	}
	for _, alias := range rule.OSAliases[os] {
		if alias == value {
			return true
		}
	}
	return false
}

// checkConstraints determines whether build constraints are satisfied on
// a given platform.
//
// The first few arguments describe the platform. genericTags is the set
// of build tags that are true on all platformConstraints. os and arch are the platform
// GOOS and GOARCH strings. If os or arch is empty, checkConstraints will
// return false in the presence of OS and architecture constraints, even
// if they are negated.
//
// The remaining arguments describe the file being tested. All of these may
// be empty or nil. osSuffix and archSuffix are filename suffixes. tags
// is the parsed build tags found near the top of the file. cgoTags
// is an extra set of tags in a #cgo directive.
func checkConstraints(c *config.Config, os, arch, osSuffix, archSuffix string, tags *buildTags, cgoTags *cgoTagsAndOpts) bool {
	if osSuffix != "" && !matchesOS(os, osSuffix) || archSuffix != "" && archSuffix != arch {
		return false
	}

	goConf := getGoConfig(c)
	checker := func(tag string, ts tagSet) bool {
		if isIgnoredTag(tag) {
			return true
		}
		if _, ok := rule.KnownOSSet[tag]; ok || tag == "unix" {
			if os == "" {
				return false
			}
			return matchesOS(os, tag)
		}

		if _, ok := rule.KnownArchSet[tag]; ok {
			if arch == "" {
				return false
			}
			return arch == tag
		}

		return ts[tag]
	}

	for _, ts := range goConf.genericTags {
		c := func(tag string) bool { return checker(tag, ts) }
		if tags.eval(c) && cgoTags.eval(c) {
			return true
		}
	}
	return false
}

// rulesGoSupportsOS returns whether the os tag is recognized by the version of
// rules_go being used. This avoids incompatibility between new versions of
// Gazelle and old versions of rules_go.
func rulesGoSupportsOS(v version.Version, os string) bool {
	if len(v) == 0 {
		return true
	}
	if v.Compare(version.Version{0, 23, 0}) < 0 &&
		(os == "aix" || os == "illumos") {
		return false
	}
	return true
}

// rulesGoSupportsArch returns whether the arch tag is recognized by the version
// of rules_go being used. This avoids incompatibility between new versions of
// Gazelle and old versions of rules_go.
func rulesGoSupportsArch(v version.Version, arch string) bool {
	if len(v) == 0 {
		return true
	}
	if v.Compare(version.Version{0, 23, 0}) < 0 &&
		arch == "riscv64" {
		return false
	}
	return true
}

// rulesGoSupportsPlatform returns whether the os and arch tag combination is
// recognized by the version of rules_go being used. This avoids incompatibility
// between new versions of Gazelle and old versions of rules_go.
func rulesGoSupportsPlatform(v version.Version, p rule.Platform) bool {
	if len(v) == 0 {
		return true
	}
	if v.Compare(version.Version{0, 23, 0}) < 0 &&
		(p.OS == "aix" && p.Arch == "ppc64" ||
			p.OS == "freebsd" && p.Arch == "arm64" ||
			p.OS == "illumos" && p.Arch == "amd64" ||
			p.OS == "linux" && p.Arch == "riscv64" ||
			p.OS == "netbsd" && p.Arch == "arm64" ||
			p.OS == "openbsd" && p.Arch == "arm64" ||
			p.OS == "windows" && p.Arch == "arm" ||
			p.OS == "windows" && p.Arch == "arm64") {
		return false
	}
	return true
}

// parseGoEmbed parses the text following "//go:embed" to extract the glob patterns.
// It accepts unquoted space-separated patterns as well as double-quoted and back-quoted Go strings.
// This is based on a similar function in cmd/compile/internal/gc/noder.go;
// this version calculates position information as well.
//
// Copied from go/build.parseGoEmbed.
func parseGoEmbed(args string, pos token.Position) ([]fileEmbed, error) {
	trimBytes := func(n int) {
		pos.Offset += n
		pos.Column += utf8.RuneCountInString(args[:n])
		args = args[n:]
	}
	trimSpace := func() {
		trim := strings.TrimLeftFunc(args, unicode.IsSpace)
		trimBytes(len(args) - len(trim))
	}

	var list []fileEmbed
	for trimSpace(); args != ""; trimSpace() {
		var path string
		pathPos := pos
	Switch:
		switch args[0] {
		default:
			i := len(args)
			for j, c := range args {
				if unicode.IsSpace(c) {
					i = j
					break
				}
			}
			path = args[:i]
			trimBytes(i)

		case '`':
			i := strings.Index(args[1:], "`")
			if i < 0 {
				return nil, fmt.Errorf("invalid quoted string in //go:embed: %s", args)
			}
			path = args[1 : 1+i]
			trimBytes(1 + i + 1)

		case '"':
			i := 1
			for ; i < len(args); i++ {
				if args[i] == '\\' {
					i++
					continue
				}
				if args[i] == '"' {
					q, err := strconv.Unquote(args[:i+1])
					if err != nil {
						return nil, fmt.Errorf("invalid quoted string in //go:embed: %s", args[:i+1])
					}
					path = q
					trimBytes(i + 1)
					break Switch
				}
			}
			if i >= len(args) {
				return nil, fmt.Errorf("invalid quoted string in //go:embed: %s", args)
			}
		}

		if args != "" {
			r, _ := utf8.DecodeRuneInString(args)
			if !unicode.IsSpace(r) {
				return nil, fmt.Errorf("invalid quoted string in //go:embed: %s", args)
			}
		}
		list = append(list, fileEmbed{path, pathPos})
	}
	return list, nil
}
