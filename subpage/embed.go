package subpage

import (
	_ "embed"
)

//go:embed page.tmpl.html
var pageHTML string

// Ensure imports referenced from page.go stay in this package's import graph
// without polluting the embed file's symbol table.
var _ = pageHTML