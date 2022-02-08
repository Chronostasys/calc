package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type TakePtrNode struct {
	Node Node
}

func (n *TakePtrNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	v := n.Node.calc(m, f, s)
	ptr := s.block.NewAlloca(v.Type())
	s.block.NewStore(v, ptr)
	return ptr
}

type TakeValNode struct {
	Level int
	Node  Node
}

func (n *TakeValNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	v := n.Node.calc(m, f, s)

	for i := 0; i < n.Level; i++ {
		v = s.block.NewLoad(getElmType(v.Type()), v)
	}
	return v

}
