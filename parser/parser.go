package parser

import (
	"strconv"

	"github.com/Chronostasys/calculator_go/ast"

	"github.com/Chronostasys/calculator_go/lexer"
)

func interger() ast.Node {
	t, err := lexer.ScanType(lexer.TYPE_INT)
	if err != nil {
		if err == lexer.ErrTYPE {
			_, err = lexer.ScanType(lexer.TYPE_LP)
			if err != nil {
				panic(err)
			}
			i := exp()
			_, err = lexer.ScanType(lexer.TYPE_RP)
			if err != nil {
				panic(err)
			}
			return i

		}
		panic(err)
	}
	i, _ := strconv.Atoi(t)
	return &ast.NumNode{Val: i}
}

func factor() ast.Node {
	a := symbol()
	code, t, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_DIV || code == lexer.TYPE_MUL {
		b := symbol()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		code, t, eos = lexer.Scan()
	}
	if !eos {
		lexer.Retract(len(t))
	}
	return a
}

func exp() ast.Node {
	a := factor()
	code, t, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		b := factor()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		code, t, eos = lexer.Scan()
	}
	if !eos {
		lexer.Retract(len(t))
	}
	return a
}

func symbol() ast.Node {
	code, t, _ := lexer.Scan()
	if code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		return &ast.UnaryNode{Op: code, Child: interger()}
	}
	lexer.Retract(len(t))
	return interger()
}

func Parse(s string) int {
	lexer.SetInput(s)
	return exp().Calc()
}
