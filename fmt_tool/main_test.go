package main

import (
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"testing"
)

func TestAst(t *testing.T) {
	f := token.NewFileSet()
	a, err := parser.ParseFile(f, "../model/structure.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	err = printer.Fprint(os.Stdout, f, a)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWalk(t *testing.T) {
	f := token.NewFileSet()
	a, err := parser.ParseFile(f, "../model/structure.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	replaceFmtCalls(a)
}
