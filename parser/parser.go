package parser

import (
	"strconv"

	"github.com/Chronostasys/calculator_go/ast"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"

	"github.com/Chronostasys/calculator_go/lexer"
)

func number() ast.Node {
	ch := lexer.SetCheckpoint()
	code, t1, eos := lexer.Scan()
	if eos {
		panic("eos")
	}
	switch code {
	case lexer.TYPE_FLOAT:
		i, err := strconv.ParseFloat(t1, 64)
		if err != nil {
			panic(err)
		}
		return &ast.NumNode{Val: constant.NewFloat(types.Float, i)}
	case lexer.TYPE_INT:
		i, err := strconv.Atoi(t1)
		if err != nil {
			panic(err)
		}
		return &ast.NumNode{Val: constant.NewInt(types.I32, int64(i))}
	}
	lexer.GobackTo(ch)
	_, err := lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		if err == lexer.ErrTYPE {
			t, err := lexer.ScanType(lexer.TYPE_VAR)
			if err != nil {
				panic(err)
			}
			return &ast.VarNode{ID: t}
		}
	}
	i := exp()
	_, err = lexer.ScanType(lexer.TYPE_RP)
	if err != nil {
		panic(err)
	}
	return i
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
	ch := lexer.SetCheckpoint()
	code, _, eos := lexer.Scan()
	if eos {
		panic(lexer.ErrEOS)
	}
	if code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		return &ast.UnaryNode{Op: code, Child: number()}
	}
	lexer.GobackTo(ch)
	return number()
}

func assign() (n ast.Node, err error) {
	c := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(c)
		}
	}()
	id, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_ASSIGN)
	if err != nil {
		return nil, err
	}
	r := exp()
	return &ast.BinNode{
		Left:  &ast.VarNode{ID: id},
		Op:    lexer.TYPE_ASSIGN,
		Right: r,
	}, nil
}

func empty() ast.Node {
	return &ast.EmptyNode{}
}

func define() (n ast.Node, err error) {
	c := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(c)
		}
	}()
	_, err = lexer.ScanType(lexer.TYPE_RES_VAR)
	if err != nil {
		return nil, err
	}
	id, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, t, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	co, ok := lexer.IsResType(t)
	if !ok {
		panic("expect reserved type")
	}
	return &ast.DefineNode{ID: id, TP: co}, nil
}

func statement() ast.Node {
	ast, err := assign()
	if err == nil {
		return ast
	}
	ast, err = define()
	if err == nil {
		return ast
	}
	return empty()
}

func statementList() ast.Node {
	n := &ast.SLNode{}
	n.Children = append(n.Children, statement())
	_, err := lexer.ScanType(lexer.TYPE_NL)
	if err == nil {
		n.Children = append(n.Children, statementList())
	} else if err != lexer.ErrEOS {
		panic("cannot recognize as a legal statement")
	}
	return n
}
func Parse(s string) string {
	lexer.SetInput(s)
	m := ir.NewModule()
	g := m.NewGlobalDef("str", constant.NewCharArrayFromString("Hello World! %d\x00"))

	printf := m.NewFunc("printf", types.I32, ir.NewParam("formatstr", types.I8Ptr))
	printf.Sig.Variadic = true
	f := m.NewFunc("main", types.Void)
	b := f.NewBlock("")
	statementList().Calc(m, f, b)
	// e := b.NewLoad(types.NewArray(uint64(len("Hello World!")+1), types.I8), g)
	zero := constant.NewInt(types.I32, 0)
	b.NewCall(printf, constant.NewGetElementPtr(g.Typ.ElemType, g, zero, zero), zero)
	b.NewRet(nil)
	return m.String()
}
