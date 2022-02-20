package ast

import (
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type StringNode struct {
	Str    string
	onheap bool
}

func (n *StringNode) setAlloc(onheap bool) {
	n.onheap = onheap
}
func (n *StringNode) travel(f func(Node)) {
	f(n)
}

func (n *StringNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {

	ch := constant.NewCharArrayFromString(n.Str)
	var alloca value.Value
	if n.onheap {
		alloca = heapAlloc(m, s, &calcedTypeNode{ch.Type()})
	} else {
		alloca = stackAlloc(m, s, ch.Type())
	}
	s.block.NewStore(ch, alloca)
	bs := s.block.NewBitCast(alloca, types.I8Ptr)
	va, _ := ScopeMap["github.com/Chronostasys/calc/runtime"].searchVar("newstr")
	return s.block.NewCall(va.v, bs, constant.NewInt(lexer.DefaultIntType(), int64(ch.Typ.Len)))
}
