package ast

import (
	"strconv"

	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type BoolConstNode struct {
	Val bool
}

func (n *BoolConstNode) tp() TypeNode {
	return &calcedTypeNode{types.I1}
}
func (n *BoolConstNode) travel(f func(Node) bool) {
	f(n)
}

func (n *BoolConstNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return constant.NewBool(n.Val)
}

type CompareNode struct {
	Op    int
	Left  ExpNode
	Right ExpNode
}

func (b *CompareNode) tp() TypeNode {
	if _, ok := b.Left.(*NilNode); ok {
		return b.Right.tp()
	}
	return b.Left.tp()
}

func (n *CompareNode) travel(f func(Node) bool) {
	f(n)
	n.Left.travel(f)
	n.Right.travel(f)
}

type e struct {
	IntE   enum.IPred
	FloatE enum.FPred
}

var comparedic = map[int]e{
	lexer.TYPE_EQ:  {enum.IPredEQ, enum.FPredOEQ},
	lexer.TYPE_NEQ: {enum.IPredNE, enum.FPredONE},
	lexer.TYPE_LG:  {enum.IPredSGT, enum.FPredOGT},
	lexer.TYPE_LEQ: {enum.IPredSGE, enum.FPredOGE},
	lexer.TYPE_SM:  {enum.IPredSLT, enum.FPredOLT},
	lexer.TYPE_SEQ: {enum.IPredSLE, enum.FPredOLE},
}

func (n *CompareNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	l, r := loadIfVar(n.Left.calc(m, f, s), s), loadIfVar(n.Right.calc(m, f, s), s)
	hasF, re := hasFloatType(s.block, l, r)
	l, r = re[0], re[1]
	_, ok1 := r.Type().(*types.PointerType)
	if _, ok := l.Type().(*types.PointerType); ok || ok1 {
		if ok {
			l = s.block.NewPtrToInt(l, lexer.DefaultIntType())
		} else {
			_, ok := n.Left.(*NilNode)
			if !ok {
				panic("expect nil")
			}
		}
		if ok1 {
			r = s.block.NewPtrToInt(r, lexer.DefaultIntType())
		} else {
			_, ok := n.Right.(*NilNode)
			if !ok {
				panic("expect nil")
			}
		}
		return s.block.NewICmp(comparedic[n.Op].IntE,
			l,
			r,
		)
	} else if hasF {
		return s.block.NewFCmp(comparedic[n.Op].FloatE, l, r)
	} else {
		return s.block.NewICmp(comparedic[n.Op].IntE, l, r)
	}

}

var blockID = 100

type IfNode struct {
	BoolExp    Node
	Statements Node
}

func (n *IfNode) travel(f func(Node) bool) {
	f(n)
	n.BoolExp.travel(f)
	n.Statements.travel(f)
}

func (n *IfNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	blockID++
	tt := f.NewBlock(strconv.Itoa(blockID))
	n.Statements.calc(m, f, s.addChildScope(tt))
	blockID++
	end := f.NewBlock(strconv.Itoa(blockID))
	s.block.NewCondBr(loadIfVar(n.BoolExp.calc(m, f, s), s), tt, end)
	s.block = end
	if tt.Term == nil {
		tt.NewBr(end)
	}
	if s.parent.block != nil {
		end.NewBr(s.parent.block)
	}

	return zero
}

type IfElseNode struct {
	BoolExp    Node
	Statements Node
	ElSt       Node
}

func (n *IfElseNode) travel(f func(Node) bool) {
	f(n)
	n.BoolExp.travel(f)
	n.Statements.travel(f)
	n.ElSt.travel(f)
}

func (n *IfElseNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	blockID++
	tt := f.NewBlock(strconv.Itoa(blockID))
	blockID++
	tf := f.NewBlock(strconv.Itoa(blockID))
	blockID++
	end := f.NewBlock(strconv.Itoa(blockID))
	s.block.NewCondBr(loadIfVar(n.BoolExp.calc(m, f, s), s), tt, tf)
	s.block = end
	n.Statements.calc(m, f, s.addChildScope(tt))
	n.ElSt.calc(m, f, s.addChildScope(tf))
	if tt.Term == nil {
		tt.NewBr(end)
	}
	if tf.Term == nil {
		tf.NewBr(end)
	}
	if s.parent.block != nil {
		end.NewBr(s.parent.block)
	}
	return zero
}

type BoolExpNode struct {
	Op    int
	Left  ExpNode
	Right ExpNode
}

func (b *BoolExpNode) tp() TypeNode {
	if _, ok := b.Left.(*NilNode); ok {
		return b.Right.tp()
	}
	return b.Left.tp()
}
func (n *BoolExpNode) travel(f func(Node) bool) {
	f(n)
	n.Left.travel(f)
	n.Right.travel(f)
}

func (n *BoolExpNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	l, r := loadIfVar(n.Left.calc(m, f, s), s), loadIfVar(n.Right.calc(m, f, s), s)
	if n.Op == lexer.TYPE_AND {
		return s.block.NewAnd(l, r)
	} else {
		return s.block.NewOr(l, r)
	}
}

type NotNode struct {
	Bool ExpNode
}

func (n *NotNode) tp() TypeNode {
	return n.Bool.tp()
}

func (n *NotNode) travel(f func(Node) bool) {
	f(n)
	n.Bool.travel(f)
}

func (n *NotNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return s.block.NewICmp(enum.IPredEQ, loadIfVar(n.Bool.calc(m, f, s), s), constant.False)
}
