package embedsrcs

import "embed"

//go:embed *m_* n_/* all:o*
var fs embed.FS
