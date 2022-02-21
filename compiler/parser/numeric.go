package parser

import (
	"strconv"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

func (p *Parser) number() (n ast.ExpNode) {
	n, err := p.runWithCatch2Exp(p.strExp)
	if err == nil {
		return n
	}
	ch := p.lexer.SetCheckpoint()
	n, err = p.runWithCatch2Exp(p.takeValExp)
	if err == nil {
		return n
	}
	code, t1, eos := p.lexer.Scan()
	if eos {
		panic("eos")
	}
	switch code {
	case lexer.TYPE_FLOAT:
		i, err := strconv.ParseFloat(t1, 32)
		tp := types.Float
		if err != nil {
			i, err = strconv.ParseFloat(t1, 64)
			if err != nil {
				panic(err)
			}
			tp = types.Double
		}
		return &ast.NumNode{Val: constant.NewFloat(tp, i)}
	case lexer.TYPE_INT:
		i, tp, err := ParseInt(t1)
		if err != nil {
			panic(err)
		}
		return &ast.NumNode{Val: constant.NewInt(tp, i)}

	}
	p.lexer.GobackTo(ch)
	_, err = p.lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	i := p.exp()
	_, err = p.lexer.ScanType(lexer.TYPE_RP)
	if err != nil {
		panic(err)
	}
	return i
}

func (p *Parser) factor() ast.ExpNode {
	a := p.symbol()
	ch := p.lexer.SetCheckpoint()
	code, _, eos := p.lexer.Scan()
	for !eos && code == lexer.TYPE_DIV ||
		code == lexer.TYPE_MUL || code == lexer.TYPE_PS {
		b := p.symbol()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = p.lexer.SetCheckpoint()
		code, _, eos = p.lexer.Scan()
	}
	if !eos {
		p.lexer.GobackTo(ch)
	}
	return a
}

func (p *Parser) exp() ast.ExpNode {
	a := p.addedFactor()
	ch := p.lexer.SetCheckpoint()
	code, _, eos := p.lexer.Scan()
	for !eos && code == lexer.TYPE_SHL || code == lexer.TYPE_SHR {
		b := p.addedFactor()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = p.lexer.SetCheckpoint()
		code, _, eos = p.lexer.Scan()
	}
	if !eos {
		p.lexer.GobackTo(ch)
	}
	return a
}
func (p *Parser) addedFactor() ast.ExpNode {
	a := p.factor()
	ch := p.lexer.SetCheckpoint()
	code, _, eos := p.lexer.Scan()
	for !eos && code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		b := p.factor()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = p.lexer.SetCheckpoint()
		code, _, eos = p.lexer.Scan()
	}
	if !eos {
		p.lexer.GobackTo(ch)
	}
	return a
}

func (p *Parser) symbol() ast.ExpNode {
	ch := p.lexer.SetCheckpoint()
	code, _, eos := p.lexer.Scan()
	if eos {
		panic(lexer.ErrEOS)
	}
	if code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		return &ast.UnaryNode{Op: code, Child: p.number()}
	}
	p.lexer.GobackTo(ch)
	return p.number()
}
