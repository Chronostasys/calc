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
	globalScope = newScope()
	typedic     = map[int]types.Type{
		lexer.TYPE_RES_FLOAT: types.Float,
		lexer.TYPE_RES_INT:   types.I32,
		lexer.TYPE_RES_BOOL:  types.I1,
	}
)

type VNode interface {
	V() value.Value
}

type Node interface {
	calc(*ir.Module, *ir.Func, *ir.Block, *scope) value.Value
}

func PrintTable() {
	fmt.Println(globalScope)
}

type BinNode struct {
	Op    int
	Left  Node
	Right Node
}

func loadIfVar(n Node, m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	if v, ok := n.(*VarNode); ok {
		l := v.calc(m, f, b, s)
		if t, ok := l.Type().(*types.PointerType); ok {
			return b.NewLoad(t.ElemType, l)
		}
	}
	return n.calc(m, f, b, s)
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

func (n *BinNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	l, r := loadIfVar(n.Left, m, f, b, s), loadIfVar(n.Right, m, f, b, s)
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
		val, err := s.searchVar(v.ID)
		if err != nil {
			panic(fmt.Errorf("variable %s not defined", v.ID))
		}
		b.NewStore(r, val)
		return val
	default:
		panic("unexpected op")
	}
}

type NumNode struct {
	Val value.Value
}

func (n *NumNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	return n.Val
}

type UnaryNode struct {
	Op    int
	Child Node
}

var zero = constant.NewInt(types.I32, 0)

func (n *UnaryNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	c := loadIfVar(n.Child, m, f, b, s)
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

func (n *VarNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	v, err := s.searchVar(n.ID)
	if err != nil {
		panic(fmt.Errorf("variable %s not defined", n.ID))
	}
	return v
}

// SLNode statement list node
type SLNode struct {
	Children []Node
}

func (n *SLNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	for _, v := range n.Children {
		v.calc(m, f, b, s)
	}
	return zero
}

type ProgramNode struct {
	Children []Node
}

func (n *ProgramNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	for _, v := range n.Children {
		v.calc(m, f, b, s)
	}
	return zero
}
func (n *ProgramNode) Emit(m *ir.Module) value.Value {
	for _, v := range n.Children {
		v.calc(m, nil, nil, globalScope)
	}
	return zero
}

type EmptyNode struct {
}

func (n *EmptyNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
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

func (n *DefineNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	if _, err := s.searchVar(n.ID); err == nil {
		panic(fmt.Errorf("redefination of var %s", n.ID))
	}
	if tp, ok := typedic[n.TP]; ok {
		if f == nil {
			n.Val = m.NewGlobal(n.ID, tp)
		} else {
			n.Val = b.NewAlloca(tp)
			s.addVar(n.ID, n.Val)
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

func (n *ParamNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	n.Val = ir.NewParam(n.ID, typedic[n.TP])
	return n.Val
}
func (n *ParamNode) V() value.Value {
	return n.Val
}

type ParamsNode struct {
	Params []Node
}

func (n *ParamsNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {

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

func (n *FuncNode) AddtoScope() {
	psn := n.Params.(*ParamsNode)
	ps := []*ir.Param{}
	for _, v := range psn.Params {
		p := v.(*ParamNode)
		param := ir.NewParam(p.ID, typedic[p.TP])
		ps = append(ps, param)
	}
	globalScope.addVar(n.ID, ir.NewFunc(n.ID, typedic[n.RetType], ps...))
}

func (n *FuncNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	// _, err := s.searchVar(n.ID)
	// if err == nil {
	// 	panic(fmt.Sprintf("re defination of func %s", n.ID))
	// }
	psn := n.Params.(*ParamsNode)
	ps := []*ir.Param{}
	childScope := s.addChildScope()

	for _, v := range psn.Params {
		p := v.(*ParamNode)
		param := v.calc(m, f, b, s).(*ir.Param)
		childScope.addVar(p.ID, param)
		ps = append(ps, param)
	}
	fn := m.NewFunc(n.ID, typedic[n.RetType], ps...)
	n.Fn = fn
	b = fn.NewBlock("")
	n.DefaultBlock = b
	s.addVar(n.ID, n.Fn)

	n.Statements.calc(m, fn, b, childScope)
	return fn
}

type CallFuncNode struct {
	Params []Node
	ID     string
}

func (n *CallFuncNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	params := []value.Value{}
	for _, v := range n.Params {
		params = append(params, loadIfVar(v, m, f, b, s))
	}
	fn, err := s.searchVar(n.ID)
	if err != nil {
		panic(err)
	}
	return b.NewCall(fn, params...)
}

type RetNode struct {
	Exp Node
}

func (n *RetNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	b.NewRet(n.Exp.calc(m, f, b, s))
	return zero
}

type BoolConstNode struct {
	Val bool
}

func (n *BoolConstNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
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

func (n *CompareNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	l, r := loadIfVar(n.Left, m, f, b, s), loadIfVar(n.Right, m, f, b, s)
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

func (n *BoolExpNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	l, r := loadIfVar(n.Left, m, f, b, s), loadIfVar(n.Right, m, f, b, s)
	if n.Op == lexer.TYPE_AND {
		return b.NewAnd(l, r)
	} else {
		return b.NewOr(l, r)
	}
}

type NotNode struct {
	Bool Node
}

func (n *NotNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	return b.NewICmp(enum.IPredEQ, loadIfVar(n.Bool, m, f, b, s), constant.False)
}

type IfNode struct {
	BoolExp    Node
	Statements Node
}

func (n *IfNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	panic("not impl")
}

type IfElseNode struct {
	BoolExp    Node
	Statements Node
	ElSt       Node
}

func (n *IfElseNode) calc(m *ir.Module, f *ir.Func, b *ir.Block, s *scope) value.Value {
	panic("not impl")
}
