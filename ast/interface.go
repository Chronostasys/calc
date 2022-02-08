package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type interf struct {
	*types.IntType
	interfaceFuncs map[string]*FuncNode
	innerType      types.Type
	name           string
}

type interfaceDefNode struct {
	id    string
	funcs map[string]*FuncNode
}

func NewSInterfaceDefNode(id string, funcsMap map[string]*FuncNode) Node {
	n := &interfaceDefNode{id: id, funcs: funcsMap}
	defFunc := func() {
		globalScope.addStruct(n.id, &typedef{
			interf: true,
			funcs:  funcsMap,
		})
	}
	globalScope.interfaceDefFuncs = append(globalScope.interfaceDefFuncs, defFunc)
	return n

}

func (n *interfaceDefNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return zero
}
