package embedsrcs

import "embed"

//go:embed *m_* n_/*
var fs embed.FS
