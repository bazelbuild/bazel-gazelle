package typescript

import "github.com/bazelbuild/bazel-gazelle/rule"

func (_ *typescriptLang) Kinds() map[string]rule.KindInfo {
	return map[string]rule.KindInfo{}
}

func (_ *typescriptLang) Loads() []rule.LoadInfo {
	return []rule.LoadInfo{}
}
