package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type PackageNode struct {
	Name string
}

func (n *PackageNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return zero
}
