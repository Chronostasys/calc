package ast

import (
	"fmt"

	"github.com/Chronostasys/calculator_go/lexer"
)

var vartable = map[string]interface{}{}

type Node interface {
	Calc() int
}

func PrintTable() {
	fmt.Println(vartable)
}

type BinNode struct {
	Op    int
	Left  Node
	Right Node
}

func (n *BinNode) Calc() int {
	switch n.Op {
	case lexer.TYPE_PLUS:
		return n.Left.Calc() + n.Right.Calc()
	case lexer.TYPE_DIV:
		return n.Left.Calc() / n.Right.Calc()
	case lexer.TYPE_MUL:
		return n.Left.Calc() * n.Right.Calc()
	case lexer.TYPE_SUB:
		return n.Left.Calc() - n.Right.Calc()
	case lexer.TYPE_ASSIGN:
		v, ok := n.Left.(*VarNode)
		if !ok {
			panic("assign statement's left side can only be variables")
		}
		_, ext := vartable[v.ID]
		if !ext {
			panic(fmt.Errorf("variable %s not defined", v.ID))
		}
		vartable[v.ID] = n.Right.Calc()
		return vartable[v.ID].(int)
	default:
		panic("unexpected op")
	}
}

type NumNode struct {
	Val int
}

func (n *NumNode) Calc() int {
	return n.Val
}

type UnaryNode struct {
	Op    int
	Child Node
}

func (n *UnaryNode) Calc() int {
	switch n.Op {
	case lexer.TYPE_PLUS:
		return n.Child.Calc()
	case lexer.TYPE_SUB:
		return -n.Child.Calc()
	default:
		panic("unexpected op")
	}
}

type VarNode struct {
	ID string
}

func (n *VarNode) Calc() int {
	v, ok := vartable[n.ID]
	if !ok {
		panic(fmt.Errorf("variable %s not defined", n.ID))
	}
	return v.(int)
}

// SLNode statement list node
type SLNode struct {
	Children []Node
}

func (n *SLNode) Calc() int {
	for _, v := range n.Children {
		v.Calc()
	}
	return 0
}

type EmptyNode struct {
}

func (n *EmptyNode) Calc() int {
	return 0
}

type DefineNode struct {
	ID string
	TP int
}

func (n *DefineNode) Calc() int {
	if _, ok := vartable[n.ID]; ok {
		panic(fmt.Errorf("redefination of var %s", n.ID))
	}
	switch n.TP {
	case lexer.TYPE_RES_INT:
		vartable[n.ID] = 0
	default:
		panic(fmt.Errorf("unknown type code %d", n.TP))
	}
	return 0
}
