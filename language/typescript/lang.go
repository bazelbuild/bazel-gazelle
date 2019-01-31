package typescript

import (
	"github.com/bazelbuild/bazel-gazelle/language"
)

type typescriptLang struct {
}

const typescriptName = "ts"

func (_ *typescriptLang) Name() string {
	return typescriptName
}

func NewLanguage() language.Language {
	return &typescriptLang{}
}
