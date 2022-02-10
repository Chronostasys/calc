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

func NewSInterfaceDefNode(id string, funcsMap map[string]*FuncNode, s *Scope) Node {
	n := &interfaceDefNode{id: id, funcs: funcsMap}
	defFunc := func(s *Scope) {
		s.globalScope.addStruct(n.id, &typedef{
			interf: true,
			funcs:  funcsMap,
		})
	}
	s.globalScope.interfaceDefFuncs = append(s.globalScope.interfaceDefFuncs, defFunc)
	return n

}

func (n *interfaceDefNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}
