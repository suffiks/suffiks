package docparser

import (
	"log"

	"github.com/yuin/goldmark/ast"
)

func convert(n ast.Node, source []byte) (*Document, error) {
	doc := &Document{
		TokenBase: TokenBase{
			Kind: "Document",
		},
	}
	var root Token = doc

	err := ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			root = root.parent()
			if root == nil || n.Kind() == ast.KindDocument {
				return ast.WalkStop, nil
			}
			return ast.WalkContinue, nil
		}

		var tok Token
		switch n := n.(type) {
		case *ast.Document:
			return ast.WalkContinue, nil
		case *ast.Heading:
			tok = &Heading{
				TokenBase: TokenBase{
					Kind: "Heading",
				},
				Level: n.Level,
			}
		case *ast.Text:
			prev := root.lastChild()
			pt, ok := prev.(*Text)
			if ok {
				pt.Text += string(n.Text(source))
				root = prev
				return ast.WalkContinue, nil
			}

			tok = &Text{
				TokenBase: TokenBase{
					Kind: "Text",
				},
				Text: string(n.Text(source)),
			}
		case *ast.TextBlock:
			tok = &Text{
				TokenBase: TokenBase{
					Kind: "Text",
				},
				Text: string(n.Text(source)),
			}
		case *ast.Paragraph:
			tok = &Paragraph{
				TokenBase: TokenBase{
					Kind: "Paragraph",
				},
			}
		case *ast.CodeSpan:
			tok = &CodeSpan{
				TokenBase: TokenBase{
					Kind: "CodeSpan",
				},
				Text: string(n.Text(source)),
			}
		case *ast.FencedCodeBlock:
			cb := &CodeBlock{
				TokenBase: TokenBase{
					Kind: "CodeBlock",
				},
				Language: string(n.Language(source)),
			}
			l := n.Lines().Len()
			for i := 0; i < l; i++ {
				line := n.Lines().At(i)
				cb.Text += string(line.Value(source))
			}
			tok = cb
		case *ast.Link:
			tok = &Link{
				TokenBase: TokenBase{
					Kind: "Link",
				},
				Destination: string(n.Destination),
				Title:       string(n.Title),
			}

		case *ast.List:
			tok = &List{
				TokenBase: TokenBase{
					Kind: "List",
				},
				Ordered: n.IsOrdered(),
				Start:   n.Start,
				Tight:   n.IsTight,
			}

		case *ast.ListItem:
			tok = &ListItem{
				TokenBase: TokenBase{
					Kind: "ListItem",
				},
				Offset: n.Offset,
			}
		case *ASTAdmonition:
			tok = &Admonition{
				TokenBase: TokenBase{
					Kind: "Admonition",
				},
				Level: n.Level,
				Title: n.Title,
			}
		default:
			log.Printf("Unknown node kind: %T", n)
			return ast.WalkSkipChildren, nil
		}
		tok.setParent(root)
		root.addChild(tok)
		root = tok
		return ast.WalkContinue, nil
	})

	var fix func(tok Token)
	fix = func(tok Token) {
		if tt, ok := tok.(*Text); ok && len(tok.tokens()) > 0 {
			tt.Text = ""
		}
		for _, t := range tok.tokens() {
			fix(t)
		}
	}
	fix(doc)

	return doc, err
}

type Token interface {
	kind() string
	parent() Token
	setParent(Token)
	addChild(Token)
	lastChild() Token
	tokens() []Token
}

type TokenBase struct {
	Kind   string  `json:"kind"`
	Tokens []Token `json:"tokens,omitempty"`
	p      Token
}

func (t *TokenBase) kind() string {
	return t.Kind
}

func (t *TokenBase) parent() Token {
	return t.p
}

func (t *TokenBase) setParent(tok Token) {
	t.p = tok
}

func (t *TokenBase) addChild(tok Token) {
	t.Tokens = append(t.Tokens, tok)
}

func (t *TokenBase) lastChild() Token {
	if len(t.Tokens) == 0 {
		return nil
	}
	return t.Tokens[len(t.Tokens)-1]
}

func (t *TokenBase) tokens() []Token {
	return t.Tokens
}

type Document struct {
	TokenBase
}

type Heading struct {
	TokenBase
	Level int `json:"level,omitempty"`
}
type Paragraph struct {
	TokenBase
}

type Text struct {
	TokenBase
	Text string `json:"text,omitempty"`
}

type CodeSpan struct {
	TokenBase
	Text string `json:"text,omitempty"`
}

type CodeBlock struct {
	TokenBase
	Language string `json:"language,omitempty"`
	Text     string `json:"text,omitempty"`
}

type Link struct {
	TokenBase
	Destination string `json:"destination,omitempty"`
	Title       string `json:"title,omitempty"`
}

type List struct {
	TokenBase
	Start   int  `json:"start,omitempty"`
	Ordered bool `json:"ordered,omitempty"`
	Tight   bool `json:"tight,omitempty"`
}

type ListItem struct {
	TokenBase
	Offset int `json:"offset,omitempty"`
}

type Admonition struct {
	TokenBase
	Level string `json:"level,omitempty"`
	Title string `json:"title,omitempty"`
}
