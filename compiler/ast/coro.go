package ast

import (
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type AwaitNode struct {
	Exp       ExpNode
	label     string
	generator types.Type
	val       func(s *Scope) value.Value
}

func (n *AwaitNode) tp() TypeNode {
	panic("not impl")
}

func (n *AwaitNode) travel(f func(Node)) {
	f(n)
	n.Exp.travel(f)
}
func (n *AwaitNode) getID() string {
	return "generator." + n.label
}

func (n *AwaitNode) setVal(f func(s *Scope) value.Value) {
	n.val = f
}

func (n *AwaitNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	var stateMachine value.Value
	if n.val != nil {
		stateMachine = n.val(s)
		st := n.Exp.calc(m, f, s)
		store(loadIfVar(st, s), stateMachine, s)
	} else {
		stateMachine = n.Exp.calc(m, f, s)
	}
	i := ScopeMap[CORO_SM_MOD].getStruct("StateMachine").structType
	smtp := loadElmType(stateMachine.Type()).(*interf)
	n.generator = smtp
	var p value.Value
	if len(f.Params) != 0 {
		p, _ = implicitCast(f.Params[0], i, s)
	}
	if p != nil {
		// 给statemachine的nexttask赋值
		stiptr := s.block.NewGetElementPtr(smtp.Type, stateMachine,
			zero, zero)
		sti := loadIfVar(stiptr, s)
		ptr := s.block.NewIntToPtr(sti, types.NewPointer(lexer.DefaultIntType()))
		hs := gcmalloc(m, s, &calcedTypeNode{i})
		store(p, hs, s)

		store(s.block.NewPtrToInt(hs, lexer.DefaultIntType()), ptr, s)
	}
	// statemachie入队列
	vst := loadIfVar(stateMachine, s)
	qt, _ := ScopeMap[CORO_MOD].searchVar("QueueTask")
	fqt := qt.v.(*ir.Func)
	c, _ := implicitCast(vst, i, s)
	s.block.NewCall(fqt, c)
	nb := f.NewBlock(n.label)
	if s.yieldBlock != nil {
		store(constant.NewBlockAddress(f, nb), s.yieldBlock, s)
	}
	s.block.NewRet(constant.False) // 自身出队列
	f1 := smtp.interfaceFuncs["GetCurrent"]
	fni := f1.i

	fn := nb.NewGetElementPtr(smtp.Type, stateMachine, zero, constant.NewInt(types.I32, int64(fni)))
	s.block = nb
	tp := smtp.genericMaps["T"]

	stiptr := s.block.NewGetElementPtr(smtp.Type, stateMachine,
		zero, zero)
	sti := loadIfVar(stiptr, s)
	ret := nb.NewCall(nb.NewIntToPtr(loadIfVar(fn, s),
		types.NewPointer(types.NewFunc(tp, types.I8Ptr))),
		nb.NewIntToPtr(sti, types.I8Ptr))
	return ret
}
