package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func (p *Parser) typeDef() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_TYPE)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	tp, err := p.allTypes()
	if err != nil {
		return nil, err
	}
	node := ast.NewTypeDef(t, tp, p.m, p.scope)
	return node, nil
}

func (p *Parser) structType() (n ast.TypeNode, err error) {
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
	}
	return &ast.StructDefNode{Fields: fields}, nil
}

func (p *Parser) interfaceType() (n ast.TypeNode, err error) {

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
	return &ast.InterfaceDefNode{Funcs: fields}, nil
}
