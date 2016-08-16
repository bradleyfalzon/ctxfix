package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
)

func main() {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments|parser.DeclarationErrors|parser.AllErrors)

	if err != nil {
		log.Fatal(err)
	}

	for name, file := range pkgs["main"].Files {
		if !checkImports(name, file) {
			continue
		}
		fixDecls(name, file)

		buf := &bytes.Buffer{}
		if err := format.Node(buf, fset, file); err != nil {
			log.Fatal(err)
		}
		if err := ioutil.WriteFile(name, buf.Bytes(), 0666); err != nil {
			log.Fatal(err)
		}
	}
}

func vlogf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

// checkImports checks if file imports golang.org/x/net/context, and if so,
// changes it to stdlib context and returns true
func checkImports(name string, file *ast.File) bool {
	vlogf("Checking file: %s", name)
	var importsContext bool
	for _, i := range file.Imports {
		if i.Path.Value == `"golang.org/x/net/context"` {
			importsContext = true
			i.Path.Value = `"context"`
			break
		}
	}
	if !importsContext {
		vlogf(" - does not import context\n")
		return false
	}
	vlogf(" - imports context\n")
	return true
}

// fixDecls finds all function declarations, removes x/net/context references
// and rewrites body to use http.Request.Context()
func fixDecls(name string, file *ast.File) {
	for _, d := range file.Decls {
		switch decl := d.(type) {
		case *ast.FuncDecl:
			vlogf("Checking function: %s", decl.Name.Name)
			// Check if function takes a context AND a http.Request
			// if so, remove context from parameter
			// scan body for that old context and replace with request's
			// scan everything all over again to replace function args
			if decl.Type.Params.List == nil {
				// Does not have parameters
				vlogf(" - does not have parameters\n")
				break
			}
			var (
				ctxName string // name of context.Context variable in func scope
				reqName string // name of http.Request variable in func scope
				rmi     int    // index of context.Context that needs to be removed
			)
			for i, field := range decl.Type.Params.List {
				switch toString(field.Type) {
				case "context.Context":
					ctxName = field.Names[0].Name
					rmi = i
				case "*http.Request":
					reqName = field.Names[0].Name
				}
			}
			if ctxName == "" {
				vlogf(" - does not accept context.Context\n")
				break
			}
			if reqName == "" {
				vlogf(" - does not accept *http.Request (leaving britney alone)\n")
				break
			}
			vlogf(" - accepts context.Context ident %q and *http.Request as %q\n", ctxName, reqName)

			// Remove context from func parameters
			decl.Type.Params.List = append(decl.Type.Params.List[:rmi], decl.Type.Params.List[rmi+1:]...)

			// Change references from ctxIdent to reqIdent.Context()
			ast.Inspect(decl.Body, func(n ast.Node) bool {
				ident, ok := n.(*ast.Ident)
				if !ok {
					return true
				}
				if ident.Name == ctxName {
					ident.Name = reqName + ".Context()"
				}
				return true
			})
		}
	}
}

func toString(node interface{}) string {
	buf := &bytes.Buffer{}
	_ = format.Node(buf, &token.FileSet{}, node)
	return buf.String()
}
