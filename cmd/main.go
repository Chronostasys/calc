package main

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/parser"
)

func main() {
	code :=
		`
	var a int
	a = 3 + 1
	var b int
	b = a * 3

	`
	parser.Parse(code)
	ast.PrintTable()
}
