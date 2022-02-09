package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func pkgDeclare() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_PKG)
	if err != nil {
		return nil, err
	}
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, err = runWithCatch(empty)
	if err != nil {
		return nil, err
	}
	return &ast.PackageNode{Name: t}, nil
}
