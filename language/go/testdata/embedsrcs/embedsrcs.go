package embedsrcs

import "embed"

//go:embed *m_* n_/* p_dir/* all:o*
var fs embed.FS
