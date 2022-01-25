package parser

import (
	"strconv"

	"github.com/Chronostasys/calculator_go/lexer"
)

func interger() int {
	t, err := lexer.ScanType(lexer.TYPE_INT)
	if err != nil {
		if err == lexer.ErrTYPE {
			lexer.ScanType(lexer.TYPE_LP)
			i := exp()
			lexer.ScanType(lexer.TYPE_RP)
			return i

		}
		panic(err)
	}
	i, _ := strconv.Atoi(t)
	return i
}

func factor() int {
	a := interger()
	code, t, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_DIV || code == lexer.TYPE_MUL {
		b := interger()
		if code == lexer.TYPE_DIV {
			a = a / b
		} else {
			a = a * b
		}
		code, t, eos = lexer.Scan()
	}
	if !eos {
		lexer.Retract(len(t))
	}
	return a
}

func exp() int {
	a := factor()
	code, t, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		b := factor()
		if code == lexer.TYPE_PLUS {
			a = a + b
		} else {
			a = a - b
		}
		code, t, eos = lexer.Scan()
	}
	if !eos {
		lexer.Retract(len(t))
	}
	return a
}

func Parse(s string) int {
	lexer.SetInput(s)
	return exp()
}
