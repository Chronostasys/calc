package parser

import (
	"fmt"
	"strconv"

	"github.com/Chronostasys/calc/compiler/ast"
	"github.com/Chronostasys/calc/compiler/lexer"
)

func (p *Parser) allTypes() (n ast.TypeNode, err error) {
	ptrLevel := 0
	for {
		_, err = p.lexer.ScanType(lexer.TYPE_MUL)
		if err != nil {
			break
		}
		ptrLevel++
	}
	n, err = p.structType()
	if err == nil {
		goto END
	}
	n, err = p.interfaceType()
	if err == nil {
		goto END
	}
	n, err = p.basicTypes()
	if err == nil {
		goto END
	}
	n, err = p.arrayTypes()
	if err == nil {
		goto END
	}
	n, err = p.funcTypes()
	if err != nil {
		return nil, err
	}
END:
	n.SetPtrLevel(ptrLevel)
	return
}

func (p *Parser) arrayTypes() (n ast.TypeNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	var arr *ast.ArrayTypeNode = &ast.ArrayTypeNode{}
	_, err = p.lexer.ScanType(lexer.TYPE_LSB)
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.ScanType(lexer.TYPE_INT)
	if err != nil {
		return nil, err
	}
	arr.Len, _ = strconv.Atoi(t)
	_, err = p.lexer.ScanType(lexer.TYPE_RSB)
	if err != nil {
		return nil, err
	}
	if arr == nil {
		return nil, fmt.Errorf("not array type")
	}
	tn, err := p.allTypes()
	if err != nil {
		return nil, err
	}
	arr.ElmType = tn
	return arr, nil

}

func (p *Parser) basicTypes() (n ast.TypeNode, err error) {
	ch := p.lexer.SetCheckpoint()
	defer func() {
		if err != nil {
			p.lexer.GobackTo(ch)
		}
	}()
	code, t, eos := p.lexer.Scan()
	if eos {
		return nil, lexer.ErrEOS
	}
	tp := []string{t}
	co, ok := lexer.IsResType(t)
	if !ok {
		if code == lexer.TYPE_VAR {
			_, err = p.lexer.ScanType(lexer.TYPE_DOT)
			if err == nil {
				// module
				t, err = p.lexer.ScanType(lexer.TYPE_VAR)
				if err != nil {
					return nil, err
				}
				tp = append(tp, t)
				tp[0] = p.imp[tp[0]]
			}
			generic, _ := p.genericCallParams()
			return &ast.BasicTypeNode{CustomTp: tp, Generics: generic, Pkg: p.mod}, nil
		} else {
			return nil, fmt.Errorf("not basic type")
		}
	}
	return &ast.BasicTypeNode{ResType: co}, nil
}

func (p *Parser) funcTypes() (n ast.TypeNode, err error) {
	_, err = p.lexer.ScanType(lexer.TYPE_RES_FUNC)
	if err != nil {
		return nil, err
	}
	fntp := &ast.FuncTypeNode{}
	fntp.Args = p.funcParams()
	p.lexer.Peek()
	fntp.Ret, _ = p.allTypes()
	return fntp, nil
}
