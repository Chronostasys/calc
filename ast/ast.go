package ast

import (
	"fmt"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var vartable = map[string]*DefineNode{}

type Node interface {
	Calc(*ir.Module, *ir.Func, *ir.Block) value.Value
}

func PrintTable() {
	fmt.Println(vartable)
}

type BinNode struct {
	Op    int
	Left  Node
	Right Node
}

func (n *BinNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	switch n.Op {
	case lexer.TYPE_PLUS:
		return b.NewAdd(n.Left.Calc(m, f, b), n.Right.Calc(m, f, b))
	case lexer.TYPE_DIV:
		return b.NewSDiv(n.Left.Calc(m, f, b), n.Right.Calc(m, f, b))
	case lexer.TYPE_MUL:
		return b.NewMul(n.Left.Calc(m, f, b), n.Right.Calc(m, f, b))
	case lexer.TYPE_SUB:
		return b.NewSub(n.Left.Calc(m, f, b), n.Right.Calc(m, f, b))
	case lexer.TYPE_ASSIGN:
		v, ok := n.Left.(*VarNode)
		if !ok {
			panic("assign statement's left side can only be variables")
		}
		_, ext := vartable[v.ID]
		if !ext {
			panic(fmt.Errorf("variable %s not defined", v.ID))
		}
		vartable[v.ID].Val = n.Right.Calc(m, f, b)
		return vartable[v.ID].Val
	default:
		panic("unexpected op")
	}
}

type NumNode struct {
	Val value.Value
}

func (n *NumNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	return n.Val
}

type UnaryNode struct {
	Op    int
	Child Node
}

var zero = constant.NewInt(types.I32, 0)

func (n *UnaryNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	switch n.Op {
	case lexer.TYPE_PLUS:
		return n.Child.Calc(m, f, b)
	case lexer.TYPE_SUB:
		return b.NewSub(zero, n.Child.Calc(m, f, b))
	default:
		panic("unexpected op")
	}
}

type VarNode struct {
	ID string
}

func (n *VarNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	v, ok := vartable[n.ID]
	if !ok {
		panic(fmt.Errorf("variable %s not defined", n.ID))
	}
	return v.Val
}

// SLNode statement list node
type SLNode struct {
	Children []Node
}

func (n *SLNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	for _, v := range n.Children {
		v.Calc(m, f, b)
	}
	return zero
}

type EmptyNode struct {
}

func (n *EmptyNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	return zero
}

type DefineNode struct {
	ID  string
	TP  int
	Val value.Value
}

func (n *DefineNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	if _, ok := vartable[n.ID]; ok {
		panic(fmt.Errorf("redefination of var %s", n.ID))
	}
	switch n.TP {
	case lexer.TYPE_RES_INT:
		if f == nil {
			n.Val = m.NewGlobal(n.ID, types.I32)
		} else {
			n.Val = b.NewAlloca(types.I32)
		}
		vartable[n.ID] = n
	default:
		panic(fmt.Errorf("unknown type code %d", n.TP))
	}
	return n.Val
}
