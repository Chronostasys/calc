package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func heapAlloc(m *ir.Module, s *scope, gtp TypeNode) value.Value {
	gfn := globalScope.genericFuncs["heapalloc"]
	fnv := gfn(m, s, gtp)
	v := s.block.NewCall(fnv)
	return v
}

func stackAlloc(m *ir.Module, s *scope, gtp types.Type) value.Value {
	v := s.block.NewAlloca(gtp)
	gfn := globalScope.genericFuncs["zero_mem"]
	fnv := gfn(m, s, &calcedTypeNode{gtp})
	s.block.NewCall(fnv, v)
	return v
}
