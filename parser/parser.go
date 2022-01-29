package parser

import (
	"fmt"
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
		defer func() {
			if err == nil {
				empty()
			}
		}()
	}()
	id, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_ASSIGN)
	if err != nil {
		return nil, err
	}
	r := allexp()
	return &ast.BinNode{
		Left:  &ast.VarNode{ID: id},
		Op:    lexer.TYPE_ASSIGN,
		Right: r,
	}, nil
}

func empty() ast.Node {
	_, err := lexer.ScanType(lexer.TYPE_NL)
	if err != nil {
		panic(err)
	}
	return &ast.EmptyNode{}
}

func define() (n ast.Node, err error) {
	c := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(c)
		}
		defer func() {
			if err == nil {
				empty()
			}
		}()
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
	c, t, _ := lexer.Scan()
	if c == lexer.TYPE_VAR {
		lexer.GobackTo(ch)
		cf := callFunc()
		empty()
		return cf
	} else if c == lexer.TYPE_NL {
		lexer.GobackTo(ch)
		return empty()
	}
	panic(fmt.Sprintf("parse fail %s", t))
}

func statementList() ast.Node {
	n := &ast.SLNode{}
	n.Children = append(n.Children, statement())
	ch := lexer.SetCheckpoint()
	c, _, _ := lexer.Scan()
	lexer.GobackTo(ch)
	if c == lexer.TYPE_RB {
		return n
	}
	n.Children = append(n.Children, statementList())
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
		//
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
	fn.Params = append(fn.Params, allexp())
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
		fn.Params = append(fn.Params, allexp())
	}
}

func returnST() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_RET)
	if err != nil {
		return nil, err
	}
	n, err = runWithCatch2(boolexp)
	if err == nil {
		_, err = runWithCatch(empty)
		if err == nil {
			return &ast.RetNode{Exp: n}, nil
		}
	}

	n = exp()
	empty()
	return &ast.RetNode{Exp: n}, nil
}

func program() ast.Node {
	n := &ast.SLNode{}
	//
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

func allexp() ast.Node {
	n, err := runWithCatch2(boolexp)
	if err == nil {
		return n
	}

	return exp()

}

func boolexp() (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	node, err = boolean()
	if err != nil {
		return nil, err
	}
	cp := lexer.SetCheckpoint()
	co, _, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	if co == lexer.TYPE_AND || co == lexer.TYPE_OR {
		n := &ast.BoolExpNode{}
		n.Left = node
		n.Op = co
		node, err = boolexp()
		if err != nil {
			return nil, err
		}
		n.Right = node
		return n, nil
	}
	lexer.GobackTo(cp)
	return
}

func runWithCatch(f func() ast.Node) (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		lexer.GobackTo(ch)
		err = fmt.Errorf("%v", recover())
	}()
	node = f()
	return
}
func runWithCatch2(f func() (ast.Node, error)) (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		i := recover()
		if i != nil {
			err = fmt.Errorf("%v", i)
		}
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	node, err = f()
	return
}

func boolean() (node ast.Node, err error) {
	ch1 := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch1)
		}
	}()
	_, err = lexer.ScanType(lexer.TYPE_RES_TRUE)
	if err == nil {
		return &ast.BoolConstNode{Val: true}, nil
	}
	_, err = lexer.ScanType(lexer.TYPE_RES_FALSE)
	if err == nil {
		return &ast.BoolConstNode{Val: false}, nil
	}
	node, err = runWithCatch2(compare)
	if err == nil {
		return node, nil
	}
	ch := lexer.SetCheckpoint()
	code, t1, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	switch code {
	case lexer.TYPE_VAR:
		_, err := lexer.ScanType(lexer.TYPE_LP)
		if err == nil {
			lexer.GobackTo(ch)
			return callFunc(), nil
		}
		return &ast.VarNode{ID: t1}, nil
	case lexer.TYPE_NOT:
		node, err = boolean()
		if err != nil {
			return nil, err
		}
		return &ast.NotNode{Bool: node}, nil
	case lexer.TYPE_LP:
		node, err = boolexp()
		if err != nil {
			return nil, err
		}
		_, err = lexer.ScanType(lexer.TYPE_RP)
		if err != nil {
			return nil, err
		}
		return
	}

	return nil, fmt.Errorf("parse failed")
}

func compare() (node ast.Node, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	n := &ast.CompareNode{}
	n.Left = exp()
	code, _, eos := lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	switch code {
	case lexer.TYPE_EQ, lexer.TYPE_NEQ,
		lexer.TYPE_LG, lexer.TYPE_SM,
		lexer.TYPE_LEQ, lexer.TYPE_SEQ:
		n.Op = code
	default:
		return nil, fmt.Errorf("expect compare op")
	}
	n.Right = exp()
	return n, nil
}

func Parse(s string) string {
	m := ir.NewModule()
	ast.AddSTDFunc(m)
	ParseAST(s).Calc(m, nil, nil)
	return m.String()
}
func ParseAST(s string) ast.Node {
	lexer.SetInput(s)

	return program()
}
