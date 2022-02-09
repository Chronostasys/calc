package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func strExp() (n ast.Node, err error) {
	str, err := lexer.ScanType(lexer.TYPE_STR)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_PLUS)
	if err != nil {
		return nil, err
	}

	return &ast.StringNode{Str: str}, nil
}
