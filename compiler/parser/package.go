package parser

import (
	"path"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/lexer"
)

func (p *Parser) pkgDeclare() (n *ast.PackageNode, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_PKG)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, err = p.runWithCatch(p.empty)
	if err != nil {
		return nil, err
	}
	return &ast.PackageNode{Name: t}, nil
}

func (p *Parser) importStatement() (n *ast.ImportNode, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_IMPORT)
	if err != nil {
		return nil, err
	}
	str, err := p.lexer.ScanType(lexer.TYPE_STR)
	im := &ast.ImportNode{Imports: map[string]string{}}
	if err == nil {
		_, f := path.Split(str)
		im.Imports[f] = str
		return im, nil
	}
	_, err = p.lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		return nil, err
	}
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RP)
		if err == nil {
			break
		}
		_, err = p.lexer.ScanType(lexer.TYPE_NL)
		if err == nil {
			continue
		}
		str, err = p.lexer.ScanType(lexer.TYPE_STR)
		if err != nil {
			return nil, err
		}
		v, err := p.lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			_, f := path.Split(str)
			im.Imports[f] = str
		} else {
			im.Imports[v] = str
		}
	}
	return im, nil
}
