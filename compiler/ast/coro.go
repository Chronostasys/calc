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

func (n *AwaitNode) travel(f func(Node) bool) {
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
		t := loadIfVar(st, s)
		store(t, stateMachine, s)
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
	lock, _ := ScopeMap[CORO_MOD].searchVar("LockST")
	st, _ := implicitCast(loadIfVar(stateMachine, s), i, s)
	s.block.NewCall(lock.v, st)
	isdone, _ := ScopeMap[CORO_MOD].searchVar("IsDone")
	b := loadIfVar(s.block.NewCall(isdone.v, st), s)
	if p != nil {
		// 如果已经完成，则直接进行movenext；反之设置continuousTask，自己退出，在
		// await的任务完成后重新入执行队列
		ifel := &IfElseNode{
			BoolExp: &fakeNode{v: b},
		}
		tr := &fakeNode{}
		tr.f = func(m *ir.Module, f *ir.Func, s *Scope) value.Value {
			return zero
		}
		el := &fakeNode{}
		el.f = func(m *ir.Module, f *ir.Func, s *Scope) value.Value {
			// 给statemachine的nexttask赋值
			stiptr := s.block.NewGetElementPtr(smtp.Type, stateMachine,
				zero, zero)
			sti := loadIfVar(stiptr, s)
			ptr := s.block.NewIntToPtr(sti, types.NewPointer(lexer.DefaultIntType()))
			hs := gcmalloc(m, s, &calcedTypeNode{i})
			store(p, hs, s)

			store(s.block.NewPtrToInt(hs, lexer.DefaultIntType()), ptr, s)
			return zero
		}
		ifel.Statements = tr
		ifel.ElSt = el
		ifel.calc(m, f, s)
	}
	// // statemachie入队列
	// vst := loadIfVar(stateMachine, s)
	// qt, _ := ScopeMap[CORO_MOD].searchVar("QueueTask")
	// fqt := qt.v.(*ir.Func)
	// c, _ := implicitCast(vst, i, s)
	// s.block.NewCall(fqt, c)
	unlock, _ := ScopeMap[CORO_MOD].searchVar("UnLockST")
	s.block.NewCall(unlock.v, st)
	nb := f.NewBlock(n.label)
	if s.yieldBlock != nil {
		store(constant.NewBlockAddress(f, nb), s.yieldBlock, s)
	}
	s.block.NewRet(b) // 自身根据情况出队列火继续执行

	// 获取返回值
	f1 := smtp.interfaceFuncs["GetResult"]
	fni := f1.i

	fn := nb.NewGetElementPtr(smtp.Type, stateMachine, zero, constant.NewInt(types.I32, int64(fni)))
	s.block = nb
	tp := smtp.genericMaps["T"]

	stiptr := s.block.NewGetElementPtr(smtp.Type, stateMachine,
		zero, zero)
	sti := loadIfVar(stiptr, s)
	bsptr := nb.NewIntToPtr(sti, types.I8Ptr)
	ret := nb.NewCall(nb.NewIntToPtr(loadIfVar(fn, s),
		types.NewPointer(types.NewFunc(tp, types.I8Ptr))),
		bsptr)
	r := gcmalloc(m, s, &calcedTypeNode{ret.Type()})
	store(ret, r, s)

	// // free resources
	// free, _ := ScopeMap[RUNTIME].searchVar("GC_free")
	// // manually free awaited async statemachine
	// s.block.NewCall(free.v, bsptr)
	// v := stackAlloc(m, s, smtp)
	// store(loadIfVar(v, s), stateMachine, s) // free async statemachine resource
	return r
}
