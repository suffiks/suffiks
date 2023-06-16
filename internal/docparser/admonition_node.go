package docparser

import (
	"github.com/yuin/goldmark/ast"
)

var kind = ast.NewNodeKind("admonition")

type ASTAdmonition struct {
	ast.BaseBlock

	Level string
	Title string
}

var _ ast.Node = (*ASTAdmonition)(nil)

func newAdmonition(level, title string) *ASTAdmonition {
	return &ASTAdmonition{
		Level: level,
		Title: title,
	}
}

func (a *ASTAdmonition) Kind() ast.NodeKind {
	return kind
}

// Dump dumps an AST tree structure to stdout.
// This function completely aimed for debugging.
// level is a indent level. Implementer should indent informations with
// 2 * level spaces.
func (a *ASTAdmonition) Dump(source []byte, level int) {
	ast.DumpHelper(a, source, level, nil, nil)
}
