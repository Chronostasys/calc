package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type AwaitNode struct {
	Exp ExpNode
}

func (n *AwaitNode) tp() TypeNode {
	panic("not impl")
}

func (n *AwaitNode) travel(f func(Node)) {
	f(n)
	n.Exp.travel(f)
}

func (n *AwaitNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	panic("notimpl")
}
