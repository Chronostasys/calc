package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func pkgDeclare() (n *ast.PackageNode, err error) {
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

func importStatement() (n *ast.ImportNode, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_IMPORT)
	if err != nil {
		return nil, err
	}
	str, err := lexer.ScanType(lexer.TYPE_STR)
	if err == nil {
		return &ast.ImportNode{Imports: []string{str}}, nil
	}
	_, err = lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		return nil, err
	}
	im := &ast.ImportNode{}
	for {
		_, err = lexer.ScanType(lexer.TYPE_RP)
		if err == nil {
			break
		}
		_, err = lexer.ScanType(lexer.TYPE_NL)
		if err == nil {
			continue
		}
		str, err = lexer.ScanType(lexer.TYPE_STR)
		if err != nil {
			return nil, err
		}
		im.Imports = append(im.Imports, str)
	}
	return im, nil
}
