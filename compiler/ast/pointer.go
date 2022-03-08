package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type TakePtrNode struct {
	Node ExpNode
}

func (n *TakePtrNode) tp() TypeNode {
	tp := n.Node.tp().Clone()
	tp.SetPtrLevel(tp.GetPtrLevel() + 1)
	return tp
}

func (n *TakePtrNode) travel(f func(Node) bool) {
	f(n)
	n.Node.travel(f)
}

func (n *TakePtrNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	v := n.Node.calc(m, f, s)
	ptr := s.block.NewAlloca(v.Type())
	s.block.NewStore(v, ptr)
	return ptr
}

type TakeValNode struct {
	Level int
	Node  ExpNode
}

func (n *TakeValNode) travel(f func(Node) bool) {
	f(n)
	n.Node.travel(f)
}
func (n *TakeValNode) tp() TypeNode {
	tp := n.Node.tp().Clone()
	tp.SetPtrLevel(tp.GetPtrLevel() - 1)
	return tp
}

func (n *TakeValNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	v := n.Node.calc(m, f, s)

	for i := 0; i < n.Level; i++ {
		v = s.block.NewLoad(getElmType(v.Type()), v)
	}
	return v

}
