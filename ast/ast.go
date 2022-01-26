package ast

import "github.com/Chronostasys/calculator_go/lexer"

type Node interface {
	Calc() int
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
