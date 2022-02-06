package parser

import (
	"github.com/Chronostasys/calculator_go/ast"
	"github.com/Chronostasys/calculator_go/lexer"
)

func structDef() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_TYPE)
	if err != nil {
		return nil, err
	}
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]ast.TypeNode)
	_, err = lexer.ScanType(lexer.TYPE_RES_STRUCT)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			empty()
			continue
		}
		fields[t], err = allTypes()
		if err != nil {
			panic(err)
		}
		empty()
	}
	return ast.NewStructDefNode(t, fields), nil
}

func interfaceDef() (n ast.Node, err error) {
	_, err = lexer.ScanType(lexer.TYPE_RES_TYPE)
	if err != nil {
		return nil, err
	}
	t, err := lexer.ScanType(lexer.TYPE_VAR)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]*ast.FuncNode)
	_, err = lexer.ScanType(lexer.TYPE_RES_INTERFACE)
	if err != nil {
		return nil, err
	}
	_, err = lexer.ScanType(lexer.TYPE_LB)
	if err != nil {
		return nil, err
	}
	for {
		_, err = lexer.ScanType(lexer.TYPE_RB)
		if err == nil {
			break
		}
		t, err := lexer.ScanType(lexer.TYPE_VAR)
		if err != nil {
			empty()
			continue
		}
		fields[t] = &ast.FuncNode{}
		fields[t].Params = funcParams()

		fields[t].RetType, err = allTypes()
		if err != nil {
			panic(err)
		}
		empty()
	}
	return ast.NewSInterfaceDefNode(t, fields), nil
}
