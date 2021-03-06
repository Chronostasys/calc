package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func gcmalloc(m *ir.Module, s *Scope, gtp TypeNode) value.Value {
	gfn := s.globalScope.getGenericFunc("heapalloc")
	if gfn == nil {
		gfn = ScopeMap["github.com/Chronostasys/calc/runtime"].getGenericFunc("heapalloc")
	}
	fnv := gfn(m, gtp)
	v := s.block.Parent.Blocks[0].NewCall(fnv)
	return v
}
func malloc(m *ir.Module, s *Scope, gtp TypeNode) value.Value {
	gfn := s.globalScope.getGenericFunc("heapmalloc")
	if gfn == nil {
		gfn = ScopeMap["github.com/Chronostasys/calc/runtime"].getGenericFunc("heapmalloc")
	}
	fnv := gfn(m, gtp)
	v := s.block.NewCall(fnv)
	return v
}

func stackAlloc(m *ir.Module, s *Scope, gtp types.Type) value.Value {
	v := s.block.Parent.Blocks[0].NewAlloca(gtp)
	return v
}
