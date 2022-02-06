package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func extFuncParam() (n *ast.ParamNode, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_THIS)
	if err != nil {
		return nil, err
	}
	n = funcParam()
	return
}

func funcParam() *ast.ParamNode {
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		panic(err)
	}
	tp, err := allTypes()
	if err != nil {
		panic(err)
	}
	return &ast.ParamNode{ID: t, TP: tp}
}

func funcParams() *ast.ParamsNode {
	_, err := lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	_, err = lexer.ScanType(lexer.TYPE_RP)
	if err == nil {
		return &ast.ParamsNode{Params: []*ast.ParamNode{}}
	}
	if err == lexer.ErrEOS {
		panic(err)
	}
	pn := &ast.ParamsNode{}
	n, err := extFuncParam()
	if err != nil {
		pn.Params = append(pn.Params, funcParam())
	} else {
		pn.Params = append(pn.Params, n)
		pn.Ext = true
	}
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
	fn.Generics, _ = genericParams()
	fn.Params = funcParams()
	if fn.Params.Ext {
		fn.ID = fn.Params.Params[0].TP.String() + "." + fn.ID
	}
	tp, err := allTypes()
	if err != nil {
		panic(err)
	}
	fn.RetType = tp
	fn.Statements, err = statementBlock()
	if err != nil {
		panic(err)
	}
	fn.AddtoScope()
	return fn
}

func callFunc() ast.Node {
	fnnode, err := runWithCatch2(varChain)
	if err != nil {
		panic(err)
	}
	fn := &ast.CallFuncNode{FnNode: fnnode}
	fn.Generics, _ = genericCallParams()
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
	_, err = runWithCatch(empty)
	if err == nil {
		return &ast.RetNode{}, nil
	}
	return &ast.RetNode{Exp: allexp()}, nil
}

func genericParams() (n []string, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	_, err = lexer.ScanType(lexer.TYPE_SM)
	if err != nil {
		return nil, err
	}
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	n = append(n, t)

	for {
		_, err = lexer.ScanType(lexer.TYPE_LG)
		if err == nil {
			return n, nil
		}
		t, err := lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			return nil, err
		}
		n = append(n, t)
	}
}

func genericCallParams() (n []ast.TypeNode, err error) {
	ch := lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			lexer.GobackTo(ch)
		}
	}()
	_, err = lexer.ScanType(lexer.TYPE_SM)
	if err != nil {
		return nil, err
	}
	t, err := allTypes()
	if err != nil {
		return nil, err
	}
	n = append(n, t)

	for {
		_, err = lexer.ScanType(lexer.TYPE_LG)
		if err == nil {
			return n, nil
		}
		_, err = lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			return nil, err
		}
		t, err := allTypes()
		if err != nil {
			return nil, err
		}
		n = append(n, t)
	}
}
