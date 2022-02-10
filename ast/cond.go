package ast

import (
	"strconv"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type BoolConstNode struct {
	Val bool
}

func (n *BoolConstNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return constant.NewBool(n.Val)
}

type CompareNode struct {
	Op    int
	Left  Node
	Right Node
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

func (n *IfNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	blockID++
	tt := f.NewBlock(strconv.Itoa(blockID))
	n.Statements.calc(m, f, s.addChildScope(tt))
	blockID++
	end := f.NewBlock(strconv.Itoa(blockID))
	s.block.NewCondBr(n.BoolExp.calc(m, f, s), tt, end)
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

func (n *IfElseNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	blockID++
	tt := f.NewBlock(strconv.Itoa(blockID))
	blockID++
	tf := f.NewBlock(strconv.Itoa(blockID))
	blockID++
	end := f.NewBlock(strconv.Itoa(blockID))
	s.block.NewCondBr(n.BoolExp.calc(m, f, s), tt, tf)
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
	Left  Node
	Right Node
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
	Bool Node
}

func (n *NotNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return s.block.NewICmp(enum.IPredEQ, loadIfVar(n.Bool.calc(m, f, s), s), constant.False)
}
