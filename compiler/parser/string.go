package parser

import (
	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/lexer"
)

func (p *Parser) strExp() (n ast.Node, err error) {
	str, err := p.lexer.ScanType(lexer.TYPE_STR)
	if err != nil {
		return nil, err
	}

	return &ast.StringNode{Str: str}, nil
}
