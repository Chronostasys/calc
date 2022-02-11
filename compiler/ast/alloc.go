package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func heapAlloc(m *ir.Module, s *Scope, gtp TypeNode) value.Value {
	gfn := s.globalScope.getGenericFunc("heapalloc")
	fnv := gfn(m, gtp)
	v := s.block.NewCall(fnv)
	return v
}

func stackAlloc(m *ir.Module, s *Scope, gtp types.Type) value.Value {
	v := s.block.NewAlloca(gtp)
	gfn := s.globalScope.getGenericFunc("zero_mem")
	fnv := gfn(m, &calcedTypeNode{gtp})
	s.block.NewCall(fnv, v)
	return v
}
