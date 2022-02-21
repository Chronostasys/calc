package ast

import (
	"strconv"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type ForNode struct {
	Bool         Node
	DefineAssign Node
	Assign       Node
	Statements   Node
}

func (n *ForNode) travel(f func(Node)) {
	f(n)
	if n.Bool != nil {
		n.Bool.travel(f)
	}
	if n.DefineAssign != nil {
		n.DefineAssign.travel(f)
	}
	if n.Assign != nil {
		n.Assign.travel(f)
	}

	n.Statements.travel(f)
}

func (n *ForNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	blockID++
	cond := f.NewBlock(strconv.Itoa(blockID))
	blockID++
	body := f.NewBlock(strconv.Itoa(blockID))
	blockID++
	end := f.NewBlock(strconv.Itoa(blockID))
	s.continueBlock = cond
	s.breakBlock = end
	child := s.addChildScope(body)
	condScope := s.addChildScope(cond)
	name := ""
	if n.DefineAssign != nil {
		n.DefineAssign.calc(m, f, s)
		name = n.DefineAssign.(*DefAndAssignNode).ID
	}
	if n.Bool != nil {
		s.block.NewCondBr(loadIfVar(n.Bool.calc(m, f, s), s), body, end)
	} else {
		s.block.NewBr(body)
	}
	s.block = end
	n.Statements.calc(m, f, child)
	if n.Assign != nil {
		n.Assign.calc(m, f, condScope)
	}
	if n.Bool != nil {
		cond.NewCondBr(loadIfVar(n.Bool.calc(m, f, condScope), condScope), body, end)
	} else {
		cond.NewBr(body)
	}
	child.block.NewBr(cond)
	if n.DefineAssign != nil {
		// a trick, ensure loop var cannot be use out of loop
		child.vartable[name] = s.vartable[name]
		delete(s.vartable, name)
	}
	return zero
}

type BreakNode struct {
}

func (n *BreakNode) travel(f func(Node)) {
	f(n)
}

func (n *BreakNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if s.breakBlock == nil {
		panic("cannot break out of loop")
	}
	s.block.NewBr(s.breakBlock)
	return zero
}

type ContinueNode struct {
}

func (n *ContinueNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if s.continueBlock == nil {
		panic("cannot continue out of loop")
	}
	s.block.NewBr(s.continueBlock)
	return zero
}

func (n *ContinueNode) travel(f func(Node)) {
	f(n)
}
