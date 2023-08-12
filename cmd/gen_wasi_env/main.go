package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, filepath.Join("internal/waruntime"), nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	if pkgs["waruntime"] == nil {
		panic("package not found")
	}

	wr := pkgs["waruntime"]

	funcs := make(map[string]*ast.FuncDecl)

	receiverName := func(f *ast.FuncDecl) string {
		if f.Recv == nil {
			return ""
		}

		str, ok := f.Recv.List[0].Type.(*ast.StarExpr)
		if ok {
			return str.X.(*ast.Ident).Name
		}

		id, ok := f.Recv.List[0].Type.(*ast.Ident)
		if ok {
			return id.Name
		}

		return ""
	}

	ast.Inspect(wr, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Package, *ast.File:
			return true
		case *ast.FuncDecl:
			recvName := receiverName(n)
			if recvName == "Runner" {
				funcs[n.Name.Name] = n
			}
		}
		return false
	})

	env, ok := funcs["env"]
	if !ok {
		panic("env not found")
	}

	decls := parseEnv(wr, funcs, env)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(decls); err != nil {
		panic(err)
	}
}

func parseEnv(wr *ast.Package, funcs map[string]*ast.FuncDecl, f *ast.FuncDecl) []decl {
	b1 := f.Body.List[0]
	if b1 == nil {
		panic("body is empty")
	}

	rs := b1.(*ast.ReturnStmt)
	els := rs.Results[0].(*ast.CompositeLit).Elts

	decls := []decl{}
	for _, el := range els {
		kv := el.(*ast.KeyValueExpr)
		key, _ := strconv.Unquote(kv.Key.(*ast.BasicLit).Value)
		val := kv.Value.(*ast.SelectorExpr).Sel.Name

		decls = append(decls, genDecl(key, val, funcs[val]))
	}

	return decls
}

type arg struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type"`
}

type decl struct {
	Name   string `json:"name"`
	Doc    string `json:"doc"`
	Args   []arg  `json:"args"`
	Return []arg  `json:"return"`
}

func genDecl(name string, target string, f *ast.FuncDecl) decl {
	d := decl{
		Name:   name,
		Doc:    strings.TrimSpace(f.Doc.Text()),
		Args:   []arg{},
		Return: []arg{},
	}

	for _, ar := range f.Type.Params.List {
		typ := paramType(ar)
		if typ == "" {
			continue
		}
		for _, name := range ar.Names {
			d.Args = append(d.Args, arg{
				Name: name.Name,
				Type: typ,
			})
		}
	}

	if f.Type.Results != nil {
		for _, ar := range f.Type.Results.List {
			typ := paramType(ar)
			if typ == "" {
				continue
			}
			d.Return = append(d.Return, arg{
				Type: typ,
			})
		}
	}

	return d
}

func paramType(p *ast.Field) string {
	switch p.Type.(type) {
	case *ast.Ident:
		return p.Type.(*ast.Ident).Name
	case *ast.StarExpr:
		return "*" + p.Type.(*ast.StarExpr).X.(*ast.Ident).Name
	}
	return ""
}
