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
		i, err := strconv.ParseFloat(t1, 32)
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
	case lexer.TYPE_VAR:
		_, err := lexer.ScanType(lexer.TYPE_LP)
		if err == nil {
			lexer.GobackTo(ch)
			return callFunc()
		}

		return &ast.VarNode{ID: t1}
	}
	lexer.GobackTo(ch)
	_, err := lexer.ScanType(lexer.TYPE_LP)
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

func factor() ast.Node {
	a := symbol()
	ch := lexer.SetCheckpoint()
	code, _, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_DIV || code == lexer.TYPE_MUL {
		b := symbol()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = lexer.SetCheckpoint()
		code, _, eos = lexer.Scan()
	}
	if !eos {
		lexer.GobackTo(ch)
	}
	return a
}

func exp() ast.Node {
	a := factor()
	ch := lexer.SetCheckpoint()
	code, _, eos := lexer.Scan()
	for !eos && code == lexer.TYPE_PLUS || code == lexer.TYPE_SUB {
		b := factor()
		a = &ast.BinNode{
			Op:    code,
			Left:  a,
			Right: b,
		}
		ch = lexer.SetCheckpoint()
		code, _, eos = lexer.Scan()
	}
	if !eos {
		lexer.GobackTo(ch)
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
	ast, err = returnST()
	if err == nil {
		return ast
	}
	ch := lexer.SetCheckpoint()
	c, _, _ := lexer.Scan()
	lexer.GobackTo(ch)
	if c == lexer.TYPE_VAR {
		return callFunc()
	}
	return empty()
}

func statementList() ast.Node {
	n := &ast.SLNode{}
	n.Children = append(n.Children, statement())
	_, err := lexer.ScanType(lexer.TYPE_NL)
	if err == nil {
		ch := lexer.SetCheckpoint()
		c, _, _ := lexer.Scan()
		lexer.GobackTo(ch)
		if c == lexer.TYPE_RB {
			return n
		}
		n.Children = append(n.Children, statementList())
	} else if err != lexer.ErrEOS {
		panic("cannot recognize as a legal statement")
	}
	return n
}

func funcParam() ast.Node {
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		panic(err)
	}
	_, tp, eos := lexer.Scan()
	if eos {
		panic(lexer.ErrEOS)
	}
	co, ok := lexer.IsResType(tp)
	if !ok {
		panic("expect reserved type")
	}
	return &ast.ParamNode{ID: t, TP: co}
}

func funcParams() ast.Node {
	_, err := lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	_, err = lexer.ScanType(lexer.TYPE_RP)
	if err == nil {
		return &ast.ParamsNode{Params: []ast.Node{}}
	}
	if err == lexer.ErrEOS {
		panic(err)
	}
	pn := &ast.ParamsNode{}
	pn.Params = append(pn.Params, funcParam())
	for {
		_, err = lexer.ScanType(lexer.TYPE_RP)
		if err == nil {
			return pn
		}
		if err == lexer.ErrEOS {
			panic(err)
		}
		_, err = lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			panic(err)
		}
		pn.Params = append(pn.Params, funcParam())
	}
}

func function() ast.Node {
	_, err := lexer.ScanType(lexer.TYPE_RES_FUNC)
	if err != nil {
		// lexer.PrintPos()
		panic(err)
	}
	id, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		panic(err)
	}
	fn := &ast.FuncNode{ID: id}
	fn.Params = funcParams()
	_, tp, eos := lexer.Scan()
	if eos {
		panic(lexer.ErrEOS)
	}
	co, ok := lexer.IsResType(tp)
	if !ok {
		panic("expect reserved type")
	}
	fn.RetType = co
	_, err = lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		panic(err)
	}
	fn.Statements = statementList()
	_, err = lexer.ScanType(lexer.TYPE_RB)
	if err != nil {
		panic(err)
	}
	return fn
}

func callFunc() ast.Node {
	getvar := func() ast.Node {
		t, err := lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			panic(err)
		}
		return &ast.VarNode{ID: t}
	}
	id, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		panic(err)
	}
	fn := &ast.CallFuncNode{ID: id}
	_, err = lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	_, err = lexer.ScanType(lexer.TYPE_RP)
	if err == nil {
		return fn
	}
	if err == lexer.ErrEOS {
		panic(err)
	}
	fn.Params = append(fn.Params, getvar())
	for {
		_, err = lexer.ScanType(lexer.TYPE_RP)
		if err == nil {
			return fn
		}
		if err == lexer.ErrEOS {
			panic(err)
		}
		_, err = lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			panic(err)
		}
		fn.Params = append(fn.Params, getvar())
	}
}

func returnST() (ast.Node, error) {
	_, err := lexer.ScanType(lexer.TYPE_RES_RET)
	if err != nil {
		return nil, err
	}
	return &ast.RetNode{Exp: exp()}, nil
}

func program() ast.Node {
	n := &ast.SLNode{}
	n.Children = append(n.Children, function())
	// lexer.PrintPos()
	for {
		ch := lexer.SetCheckpoint()
		c, _, eos := lexer.Scan()
		if c == lexer.TYPE_NL {
			continue
		}
		lexer.GobackTo(ch)
		if eos {
			break
		}
		n.Children = append(n.Children, function())
	}
	return n
}

func Parse(s string) string {
	lexer.SetInput(s)
	m := ir.NewModule()
	ast.AddSTDFunc(m)
	program().Calc(m, nil, nil)
	return m.String()
}
