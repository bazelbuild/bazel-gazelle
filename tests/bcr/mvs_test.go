package main

import (
	"testing"

	toml "github.com/pelletier/go-toml"
	"github.com/stretchr/testify/require"
)

func TestMvs(t *testing.T) {
	// Depend on functionality added in go-toml 1.8.1, which is a dependency of the 'gazelle' module, but strictly newer
	// than the version that the root module depends on, to verify that MVS is used to resolve dependencies.
	// Don't try this at home, it's not good style.
	// https://github.com/pelletier/go-toml/commit/bcacc71a18be4b36b52baf6d10a95513f94dc7b2
	tree, err := toml.Load(`[modules]
deps = ["gazelle", "rules_go"]`)
	require.NoError(t, err)
	require.Equal(t, []string{"gazelle", "rules_go"}, tree.GetArray("modules.deps"))
}
