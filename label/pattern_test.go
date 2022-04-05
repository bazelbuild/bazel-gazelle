package label

import (
	"reflect"
	"testing"
)

func TestPatternParse(t *testing.T) {
	for _, tc := range []struct {
		s       string
		want    Pattern
		wantErr bool
	}{
		{s: "//:foo", want: Pattern{SpecificName: "foo"}},
		{s: "//:bar", want: Pattern{SpecificName: "bar"}},
		{s: "//:all", want: Pattern{IsExplicitAll: true}},
		{s: "//:*", want: Pattern{IsExplicitAll: true}},
		{s: "//:all-targets", want: Pattern{IsExplicitAll: true}},
		{s: "//...", want: Pattern{Recursive: true}},
		{s: "//...:all", want: Pattern{Recursive: true, IsExplicitAll: true}},
		{s: "//...:*", want: Pattern{Recursive: true, IsExplicitAll: true}},
		{s: "//...:all-targets", want: Pattern{Recursive: true, IsExplicitAll: true}},

		{s: "//foo", want: Pattern{Pkg: "foo", SpecificName: "foo"}},
		{s: "//foo:foo", want: Pattern{Pkg: "foo", SpecificName: "foo"}},
		{s: "//foo:bar", want: Pattern{Pkg: "foo", SpecificName: "bar"}},
		{s: "//foo:all", want: Pattern{Pkg: "foo", IsExplicitAll: true}},
		{s: "//foo:*", want: Pattern{Pkg: "foo", IsExplicitAll: true}},
		{s: "//foo:all-targets", want: Pattern{Pkg: "foo", IsExplicitAll: true}},
		{s: "//foo/...", want: Pattern{Pkg: "foo", Recursive: true}},
		{s: "//foo/...:all", want: Pattern{Pkg: "foo", Recursive: true, IsExplicitAll: true}},
		{s: "//foo/...:*", want: Pattern{Pkg: "foo", Recursive: true, IsExplicitAll: true}},
		{s: "//foo/...:all-targets", want: Pattern{Pkg: "foo", Recursive: true, IsExplicitAll: true}},

		{s: "@beep//foo", want: Pattern{Repo: "beep", Pkg: "foo", SpecificName: "foo"}},
		{s: "@beep//foo:foo", want: Pattern{Repo: "beep", Pkg: "foo", SpecificName: "foo"}},
		{s: "@beep//foo:bar", want: Pattern{Repo: "beep", Pkg: "foo", SpecificName: "bar"}},
		{s: "@beep//foo:all", want: Pattern{Repo: "beep", Pkg: "foo", IsExplicitAll: true}},
		{s: "@beep//foo:*", want: Pattern{Repo: "beep", Pkg: "foo", IsExplicitAll: true}},
		{s: "@beep//foo:all-targets", want: Pattern{Repo: "beep", Pkg: "foo", IsExplicitAll: true}},
		{s: "@beep//foo/...", want: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}},
		{s: "@beep//foo/...:all", want: Pattern{Repo: "beep", Pkg: "foo", Recursive: true, IsExplicitAll: true}},
		{s: "@beep//foo/...:*", want: Pattern{Repo: "beep", Pkg: "foo", Recursive: true, IsExplicitAll: true}},
		{s: "@beep//foo/...:all-targets", want: Pattern{Repo: "beep", Pkg: "foo", Recursive: true, IsExplicitAll: true}},

		{s: ":foo", want: Pattern{SpecificName: "foo"}},
		{s: ":bar", want: Pattern{SpecificName: "bar"}},
		{s: ":all", want: Pattern{IsExplicitAll: true}},
		{s: ":*", want: Pattern{IsExplicitAll: true}},
		{s: ":all-targets", want: Pattern{IsExplicitAll: true}},
		{s: "...", want: Pattern{Recursive: true}},
		{s: "...:all", want: Pattern{Recursive: true, IsExplicitAll: true}},
		{s: "...:*", want: Pattern{Recursive: true, IsExplicitAll: true}},
		{s: "...:all-targets", want: Pattern{Recursive: true, IsExplicitAll: true}},
	} {
		got, err := ParsePattern(tc.s)
		if err != nil && !tc.wantErr {
			t.Errorf("ParsePattern(%s): got error %s ; want success", tc.s, err)
			continue
		}
		if err == nil && tc.wantErr {
			t.Errorf("ParsePattern(%s): got pattern %#v ; want error", tc.s, got)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ParsePattern(%s): got pattern %#v ; want %#v", tc.s, got, tc.want)
		}
	}
}

func TestPatternMatches(t *testing.T) {
	for _, tc := range []struct {
		pattern Pattern
		label   Label
		want    bool
	}{
		// //foo/...
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: true},
		{pattern: Pattern{Pkg: "foo", Recursive: true}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// //foo:all
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "foo", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "foo", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Pkg: "foo", IsExplicitAll: true}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// //foo:foo
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "foo", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "foo", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "foo"}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// //foo:bar
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "foo", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "foo", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Pkg: "foo", SpecificName: "bar"}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// //...
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "foo", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "foo", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: true},
		{pattern: Pattern{Pkg: "", Recursive: true}, label: Label{Pkg: "bar", Name: "baz"}, want: true},

		// //:all
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "foo", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "foo", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Pkg: "", IsExplicitAll: true}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// //:foo
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "", Name: "foo"}, want: true},
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "foo", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "foo", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "foo"}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// //:bar
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "", Name: "bar"}, want: true},
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "foo", Name: "foo"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "foo", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Pkg: "", SpecificName: "bar"}, label: Label{Pkg: "bar", Name: "baz"}, want: false},

		// @beep//foo/...
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo", Name: "foo"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo", Name: "bar"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo/bar", Name: "bar"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "foo/bar", Name: "baz"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Pkg: "bar", Name: "baz"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "", Name: "foo"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "", Name: "bar"}, want: false},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "foo", Name: "foo"}, want: true},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "foo", Name: "bar"}, want: true},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "foo/bar", Name: "bar"}, want: true},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "foo/bar", Name: "baz"}, want: true},
		{pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}, label: Label{Repo: "beep", Pkg: "bar", Name: "baz"}, want: false},
	} {
		got := tc.pattern.Matches(tc.label)
		if tc.want != got {
			t.Errorf("%#v got %v", tc, got)
		}
	}
}

func TestPatternString(t *testing.T) {
	for _, tc := range []struct {
		want    string
		pattern Pattern
	}{
		{want: "//:foo", pattern: Pattern{SpecificName: "foo"}},
		{want: "//:bar", pattern: Pattern{SpecificName: "bar"}},
		{want: "//:all", pattern: Pattern{IsExplicitAll: true}},
		{want: "//...", pattern: Pattern{Recursive: true}},
		{want: "//...:all", pattern: Pattern{Recursive: true, IsExplicitAll: true}},

		{want: "//foo", pattern: Pattern{Pkg: "foo", SpecificName: "foo"}},
		{want: "//foo:bar", pattern: Pattern{Pkg: "foo", SpecificName: "bar"}},
		{want: "//foo:all", pattern: Pattern{Pkg: "foo", IsExplicitAll: true}},
		{want: "//foo/...", pattern: Pattern{Pkg: "foo", Recursive: true}},
		{want: "//foo/...:all", pattern: Pattern{Pkg: "foo", Recursive: true, IsExplicitAll: true}},

		{want: "@beep//foo", pattern: Pattern{Repo: "beep", Pkg: "foo", SpecificName: "foo"}},
		{want: "@beep//foo:bar", pattern: Pattern{Repo: "beep", Pkg: "foo", SpecificName: "bar"}},
		{want: "@beep//foo:all", pattern: Pattern{Repo: "beep", Pkg: "foo", IsExplicitAll: true}},
		{want: "@beep//foo/...", pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true}},
		{want: "@beep//foo/...:all", pattern: Pattern{Repo: "beep", Pkg: "foo", Recursive: true, IsExplicitAll: true}},
	} {
		got := tc.pattern.String()
		if tc.want != got {
			t.Errorf("%#v.String() = %q; want %q", tc.pattern, got, tc.want)
		}
	}
}
