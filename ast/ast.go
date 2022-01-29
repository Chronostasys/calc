package ast

import (
	"fmt"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var (
	vartable = map[string]map[string]VNode{}
	fntable  = map[string]*FuncNode{}
	typedic  = map[int]types.Type{
		lexer.TYPE_RES_FLOAT: types.Float,
		lexer.TYPE_RES_INT:   types.I32,
		lexer.TYPE_RES_BOOL:  types.I1,
	}
)

type VNode interface {
	V() value.Value
}

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

func loadIfVar(n Node, m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	if v, ok := n.(*VarNode); ok {
		l := v.Calc(m, f, b)
		if t, ok := l.Type().(*types.PointerType); ok {
			return b.NewLoad(t.ElemType, l)
		}
	}
	return n.Calc(m, f, b)
}

func hasFloatType(ts ...value.Value) bool {
	hasfloat := false
	for _, v := range ts {
		_, ok := v.Type().(*types.FloatType)
		if ok {
			hasfloat = true
			return hasfloat
		}
	}
	return hasfloat
}

func (n *BinNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	l, r := loadIfVar(n.Left, m, f, b), loadIfVar(n.Right, m, f, b)
	hasF := hasFloatType(l, r)
	switch n.Op {
	case lexer.TYPE_PLUS:
		if hasF {
			return b.NewFAdd(l, r)
		}
		return b.NewAdd(l, r)
	case lexer.TYPE_DIV:
		if hasF {
			return b.NewFDiv(l, r)
		}
		return b.NewSDiv(l, r)
	case lexer.TYPE_MUL:
		if hasF {
			return b.NewFMul(l, r)
		}
		return b.NewMul(l, r)
	case lexer.TYPE_SUB:
		if hasF {
			return b.NewFSub(l, r)
		}
		return b.NewSub(l, r)
	case lexer.TYPE_ASSIGN:
		v, ok := n.Left.(*VarNode)
		if !ok {
			panic("assign statement's left side can only be variables")
		}
		_, ext := vartable[f.Name()][v.ID]
		if !ext {
			panic(fmt.Errorf("variable %s not defined", v.ID))
		}
		b.NewStore(r, vartable[f.Name()][v.ID].V())
		return vartable[f.Name()][v.ID].V()
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
	c := loadIfVar(n.Child, m, f, b)
	switch n.Op {
	case lexer.TYPE_PLUS:
		return c
	case lexer.TYPE_SUB:
		return b.NewSub(zero, c)
	default:
		panic("unexpected op")
	}
}

type VarNode struct {
	ID string
}

func (n *VarNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	v, ok := vartable[f.Name()][n.ID]
	if !ok {
		panic(fmt.Errorf("variable %s not defined", n.ID))
	}
	return v.V()
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

func (n *DefineNode) V() value.Value {
	return n.Val
}

func (n *DefineNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	if _, ok := vartable[n.ID]; ok {
		panic(fmt.Errorf("redefination of var %s", n.ID))
	}
	if tp, ok := typedic[n.TP]; ok {
		if f == nil {
			n.Val = m.NewGlobal(n.ID, tp)
		} else {
			n.Val = b.NewAlloca(tp)
			vartable[f.Name()][n.ID] = n
		}
		return n.Val
	}
	panic(fmt.Errorf("unknown type code %d", n.TP))
}

type ParamNode struct {
	ID  string
	TP  int
	Val value.Value
}

func (n *ParamNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	n.Val = ir.NewParam(n.ID, typedic[n.TP])
	return n.Val
}
func (n *ParamNode) V() value.Value {
	return n.Val
}

type ParamsNode struct {
	Params []Node
}

func (n *ParamsNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {

	return zero
}

type FuncNode struct {
	Params       Node
	ID           string
	RetType      int
	Statements   Node
	Fn           *ir.Func
	DefaultBlock *ir.Block
}

func (n *FuncNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	_, ok := fntable[n.ID]
	if ok {
		panic(fmt.Sprintf("re defination of func %s", n.ID))
	}
	psn := n.Params.(*ParamsNode)
	ps := []*ir.Param{}
	vartable[n.ID] = map[string]VNode{}
	for _, v := range psn.Params {
		p := v.(*ParamNode)
		vartable[n.ID][p.ID] = p
		ps = append(ps, v.Calc(m, f, b).(*ir.Param))
	}
	fn := m.NewFunc(n.ID, typedic[n.RetType], ps...)
	n.Fn = fn
	b = fn.NewBlock("")
	n.DefaultBlock = b
	fntable[n.ID] = n

	n.Statements.Calc(m, fn, b)
	return fn
}

type CallFuncNode struct {
	Params []Node
	ID     string
}

func (n *CallFuncNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	params := []value.Value{}
	for _, v := range n.Params {
		params = append(params, loadIfVar(v, m, f, b))
	}
	return b.NewCall(fntable[n.ID].Fn, params...)
}

type RetNode struct {
	Exp Node
}

func (n *RetNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	b.NewRet(n.Exp.Calc(m, f, b))
	return zero
}

type BoolConstNode struct {
	Val bool
}

func (n *BoolConstNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
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

func (n *CompareNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	l, r := loadIfVar(n.Left, m, f, b), loadIfVar(n.Right, m, f, b)
	hasF := hasFloatType(l, r)
	if hasF {
		return b.NewFCmp(comparedic[n.Op].FloatE, l, r)
	} else {
		return b.NewICmp(comparedic[n.Op].IntE, l, r)
	}
}

type BoolExpNode struct {
	Op    int
	Left  Node
	Right Node
}

func (n *BoolExpNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	l, r := loadIfVar(n.Left, m, f, b), loadIfVar(n.Right, m, f, b)
	if n.Op == lexer.TYPE_AND {
		return b.NewAnd(l, r)
	} else {
		return b.NewOr(l, r)
	}
}

type NotNode struct {
	Bool Node
}

func (n *NotNode) Calc(m *ir.Module, f *ir.Func, b *ir.Block) value.Value {
	return b.NewICmp(enum.IPredEQ, loadIfVar(n.Bool, m, f, b), constant.False)
}
