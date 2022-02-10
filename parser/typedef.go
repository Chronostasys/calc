package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func (p *Parser) structDef() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_TYPE)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]ast.TypeNode)
	_, err = p.lexer.ScanType(lexer.TYPE_RES_STRUCT)
	if err != nil {
		return nil, err
	}
	_, err = p.lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := p.lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			p.empty()
			continue
		}
		fields[t], err = p.allTypes()
		if err != nil {
			panic(err)
		}
		p.empty()
	}
	return ast.NewStructDefNode(t, fields, p.scope), nil
}

func (p *Parser) interfaceDef() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_TYPE)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]*ast.FuncNode)
	_, err = p.lexer.ScanType(lexer.TYPE_RES_INTERFACE)
	if err != nil {
		return nil, err
	}
	_, err = p.lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := p.lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			p.empty()
			continue
		}
		fields[t] = &ast.FuncNode{}
		fields[t].Params = p.funcParams()

		fields[t].RetType, err = p.allTypes()
		if err != nil {
			panic(err)
		}
		p.empty()
	}
	return ast.NewSInterfaceDefNode(t, fields, p.scope), nil
}
