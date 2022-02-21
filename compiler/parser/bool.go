package parser

import (
	"fmt"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/lexer"
)

func (p *Parser) boolexp() (node ast.ExpNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	node, err = p.bitOp()
	if err != nil {
		return nil, err
	}
	cp := p.lexer.SetCheckpoint()
	co, _, eos := p.lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	if co == lexer.TYPE_AND || co == lexer.TYPE_OR {
		n := &ast.BoolExpNode{}
		n.Left = node
		n.Op = co
		node, err = p.compare()
		if err != nil {
			return nil, err
		}
		n.Right = node
		return n, nil
	}
	p.lexer.GobackTo(cp)
	return
}

func (p *Parser) bitOp() (node ast.ExpNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	node, err = p.compare()
	if err != nil {
		return nil, err
	}
	for {
		check := p.lexer.SetCheckpoint()
		code, _, end := p.lexer.Scan()
		if end {
			return node, nil
		}
		switch code {
		case lexer.TYPE_BIT_OR,
			lexer.TYPE_ESP,
			lexer.TYPE_BIT_XOR:
			right, err := p.compare()
			if err != nil {
				return nil, err
			}
			node = &ast.BinNode{
				Left:  node,
				Op:    code,
				Right: right,
			}
		default:
			p.lexer.GobackTo(check)
			return node, nil

		}
	}

}

func (p *Parser) boolean() (node ast.ExpNode, err error) {
	ch1 := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch1)
		}
	}()
	_, err = p.lexer.ScanType(lexer.TYPE_RES_TRUE)
	if err == nil {
		return &ast.BoolConstNode{Val: true}, nil
	}
	_, err = p.lexer.ScanType(lexer.TYPE_RES_FALSE)
	if err == nil {
		return &ast.BoolConstNode{Val: false}, nil
	}
	node, err = p.runWithCatchExp(p.exp)
	if err == nil {
		return node, nil
	}
	node, err = p.nilExp()
	if err == nil {
		return node, nil
	}

	code, _, eos := p.lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	switch code {
	case lexer.TYPE_NOT:
		node, err = p.boolean()
		if err != nil {
			return nil, err
		}
		return &ast.NotNode{Bool: node}, nil
	case lexer.TYPE_LP:
		node, err = p.boolexp()
		if err != nil {
			return nil, err
		}
		_, err = p.lexer.ScanType(lexer.TYPE_RP)
		if err != nil {
			return nil, err
		}
		return

	}

	return nil, fmt.Errorf("parse failed")
}

func (p *Parser) compare() (node ast.ExpNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	node, err = p.runWithCatch2Exp(p.boolean)
	if err != nil {
		return nil, err
	}
	for {
		n := &ast.CompareNode{}
		check := p.lexer.SetCheckpoint()
		code, _, eos := p.lexer.Scan()
		if eos {
			return nil, lexer.ErrEOS
		}
		switch code {
		case lexer.TYPE_EQ, lexer.TYPE_NEQ,
			lexer.TYPE_LG, lexer.TYPE_SM,
			lexer.TYPE_LEQ, lexer.TYPE_SEQ:
			n.Op = code
			n.Left = node
		default:
			p.lexer.GobackTo(check)
			return node, nil
		}
		n.Right, err = p.runWithCatch2Exp(p.boolean)
		if err != nil {
			return nil, err
		}
		node = n
	}
}
