package parser

import (
	"strings"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/lexer"
)

func (p *Parser) extFuncParam() (n *ast.ParamNode, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_THIS)
	if err != nil {
		return nil, err
	}
	n = p.funcParam()
	return
}

func (p *Parser) funcParam() *ast.ParamNode {
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		panic(err)
	}
	tp, err := p.allTypes()
	if err != nil {
		panic(err)
	}
	return &ast.ParamNode{ID: t, TP: tp}
}

func (p *Parser) funcParams() *ast.ParamsNode {
	_, err := p.lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	_, err = p.lexer.ScanType(lexer.TYPE_RP)
	if err == nil {
		return &ast.ParamsNode{Params: []*ast.ParamNode{}}
	}
	if err == lexer.ErrEOS {
		panic(err)
	}
	pn := &ast.ParamsNode{}
	n, err := p.extFuncParam()
	if err != nil {
		pn.Params = append(pn.Params, p.funcParam())
	} else {
		pn.Params = append(pn.Params, n)
		pn.Ext = true
	}
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RP)
		if err == nil {
			return pn
		}
		if err == lexer.ErrEOS {
			panic(err)
		}
		_, err = p.lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			panic(err)
		}
		pn.Params = append(pn.Params, p.funcParam())
	}
}

func (p *Parser) function() ast.Node {
	_, err := p.lexer.ScanType(lexer.TYPE_RES_FUNC)
	if err != nil {
		panic(err)
	}
	id, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		panic(err)
	}
	fn := &ast.FuncNode{ID: id}
	fn.Generics, _ = p.genericParams()
	fn.Params = p.funcParams()
	if fn.Params.Ext { // 扩展方法的第一个参数
		name := fn.Params.Params[0].TP.String(p.scope)
		idx := strings.Index(name, "<") // 去掉泛型
		if idx > -1 {
			name = name[:idx]
		}
		fn.ID = name + "." + fn.ID
	}
	tp, err := p.allTypes()
	if err != nil {
		panic(err)
	}
	fn.RetType = tp
	_, err = p.lexer.ScanType(lexer.TYPE_RES_ASYNC)
	fn.Async = err == nil
	fn.Statements, _ = p.statementBlock()

	fn.AddtoScope(p.scope)
	return fn
}

func (p *Parser) callFunc() ast.ExpNode {
	fnnode, err := p.runWithCatch2Exp(p.varChain)
	if err != nil {
		panic(err)
	}
	fn := &ast.CallFuncNode{FnNode: fnnode}
	fn.Generics, _ = p.genericCallParams()
	_, err = p.lexer.ScanType(lexer.TYPE_LP)
	if err != nil {
		panic(err)
	}
	_, err = p.lexer.ScanType(lexer.TYPE_RP)
	if err == nil {
		goto END
	}
	if err == lexer.ErrEOS {
		panic(err)
	}
	fn.Params = append(fn.Params, p.allexp())
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_RP)
		if err == nil {
			goto END
		}
		if err == lexer.ErrEOS {
			panic(err)
		}
		_, err = p.lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			panic(err)
		}
		fn.Params = append(fn.Params, p.allexp())
	}
END:
	for {
		_, err := p.lexer.ScanType(lexer.TYPE_DOT)
		if err != nil {
			break
		}
		inner, err := p.runWithCatchExp(p.callFunc)
		if err != nil {
			inner, err = p.varChain()
			if err != nil {
				panic(err)
			}
		}
		fn.Next = inner
	}
	return fn
}

func (p *Parser) returnST() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_RET)
	if err != nil {
		return nil, err
	}
	_, err = p.runWithCatch(p.empty)
	if err == nil {
		return &ast.RetNode{}, nil
	}
	return &ast.RetNode{Exp: p.allexp()}, nil
}

func (p *Parser) genericParams() (n []string, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	_, err = p.lexer.ScanType(lexer.TYPE_SM)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	n = append(n, t)

	for {
		_, err = p.lexer.ScanType(lexer.TYPE_LG)
		if err == nil {
			return n, nil
		}
		t, err := p.lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			return nil, err
		}
		n = append(n, t)
	}
}

func (p *Parser) genericCallParams() (n []ast.TypeNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	_, err = p.lexer.ScanType(lexer.TYPE_SM)
	if err != nil {
		return nil, err
	}
	t, err := p.allTypes()
	if err != nil {
		return nil, err
	}
	n = append(n, t)

	for {
		_, err = p.lexer.ScanType(lexer.TYPE_LG)
		if err == nil {
			return n, nil
		}
		_, err = p.lexer.ScanType(lexer.TYPE_COMMA)
		if err != nil {
			return nil, err
		}
		t, err := p.allTypes()
		if err != nil {
			return nil, err
		}
		n = append(n, t)
	}
}

func (p *Parser) inlineFunc() (n ast.ExpNode, err error) {
	fntp, err := p.funcTypes()
	if err != nil {
		return nil, err
	}
	fn := &ast.InlineFuncNode{
		Fntype: fntp,
	}
	_, err = p.lexer.ScanType(lexer.TYPE_RES_ASYNC)
	fn.Async = err == nil
	fn.Body, err = p.runWithCatch2(p.statementBlock)
	if err != nil {
		return nil, err
	}
	return fn, nil
}

func (p *Parser) yield() (n ast.Node, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_YIELD)
	if err != nil {
		return nil, err
	}
	exp, _ := p.runWithCatchExp(p.allexp)
	p.empty()
	return &ast.YieldNode{Exp: exp}, nil
}
