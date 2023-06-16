package docparser

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type admonition struct{}

var _ parser.BlockParser = (*admonition)(nil)

func (a *admonition) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(a, 500),
		),
	)
}

// Trigger returns a list of characters that triggers Parse method of
// this parser.
// If Trigger returns a nil, Open will be called with any lines.
func (a *admonition) Trigger() []byte {
	return []byte("!!!")
}

// Open parses the current line and returns a result of parsing.
//
// Open must not parse beyond the current line.
// If Open has been able to parse the current line, Open must advance a reader
// position by consumed byte length.
//
// If Open has not been able to parse the current line, Open should returns
// (nil, NoChildren). If Open has been able to parse the current line, Open
// should returns a new Block node and returns HasChildren or NoChildren.
func (a *admonition) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	if bytes.Equal(bytes.TrimSpace(line), []byte("!!!")) {
		return nil, parser.NoChildren
	}

	res := bytes.TrimSpace(line)
	res = bytes.TrimPrefix(res, []byte("!!!"))
	res = bytes.TrimSpace(res)
	parts := strings.Split(string(res), " ")
	level := "info"
	title := ""
	if len(parts) > 0 {
		level = parts[0]
	}
	if len(parts) > 1 {
		title = strings.Join(parts[1:], " ")
	}

	reader.Advance(len(line))
	return newAdmonition(level, title), parser.HasChildren
}

// Continue parses the current line and returns a result of parsing.
//
// Continue must not parse beyond the current line.
// If Continue has been able to parse the current line, Continue must advance
// a reader position by consumed byte length.
//
// If Continue has not been able to parse the current line, Continue should
// returns Close. If Continue has been able to parse the current line,
// Continue should returns (Continue | NoChildren) or
// (Continue | HasChildren)
func (a *admonition) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	if bytes.Equal(bytes.TrimSpace(line), []byte("!!!")) {
		reader.Advance(len(line))
		return parser.Continue | parser.HasChildren
	}
	return parser.Close
}

// Close will be called when the parser returns Close.
func (a *admonition) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// nothing to do
}

// CanInterruptParagraph returns true if the parser can interrupt paragraphs,
// otherwise false.
func (a *admonition) CanInterruptParagraph() bool {
	return true
}

// CanAcceptIndentedLine returns true if the parser can open new node when
// the given line is being indented more than 3 spaces.
func (a *admonition) CanAcceptIndentedLine() bool {
	return false
}
