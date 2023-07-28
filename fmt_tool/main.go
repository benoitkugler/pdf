package main

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
)

func main() {
}

type v struct{}

func (v v) Visit(n ast.Node) ast.Visitor {
	if expr, ok := n.(*ast.CallExpr); ok {
		if s, ok := expr.Fun.(*ast.SelectorExpr); ok {
			if s.Sel.Name == "Sprintf" {
				handleSprintf(expr)
			}
		}
	}
	return v
}

var refmt = regexp.MustCompile(`%(s|d|f|g|\.\df)`)

func handleSprintf(expr *ast.CallExpr) {
	// look for the format string
	if len(expr.Args) < 2 {
		return
	}
	// we only handle string litterals
	if fmtString, ok := expr.Args[0].(*ast.BasicLit); ok {
		// we do a very basic analysis
		fmt.Println(fmtString.Value, refmt.FindAllStringSubmatch(fmtString.Value, -1))
		index := strings.Index(fmtString.Value, "%")
		if index > 0 && index < len(fmtString.Value) {
			// TODO:
		}
	}
}

func replaceFmtCalls(a *ast.File) {
	ast.Walk(v{}, a)
}
