package ast

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var (
	globalScope = newScope(nil)
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
	calc(*ir.Module, *ir.Func, *scope) value.Value
}

func PrintTable() {
	fmt.Println(globalScope)
}

type BinNode struct {
	Op    int
	Left  Node
	Right Node
}

func loadIfVar(n Node, m *ir.Module, f *ir.Func, s *scope) value.Value {
	l := n.calc(m, f, s)

	if t, ok := l.Type().(*types.PointerType); ok {
		return s.block.NewLoad(t.ElemType, l)
	}
	return n.calc(m, f, s)
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

func (n *BinNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	l, r := loadIfVar(n.Left, m, f, s), loadIfVar(n.Right, m, f, s)
	hasF := hasFloatType(l, r)
	switch n.Op {
	case lexer.TYPE_PLUS:
		if hasF {
			return s.block.NewFAdd(l, r)
		}
		return s.block.NewAdd(l, r)
	case lexer.TYPE_DIV:
		if hasF {
			return s.block.NewFDiv(l, r)
		}
		return s.block.NewSDiv(l, r)
	case lexer.TYPE_MUL:
		if hasF {
			return s.block.NewFMul(l, r)
		}
		return s.block.NewMul(l, r)
	case lexer.TYPE_SUB:
		if hasF {
			return s.block.NewFSub(l, r)
		}
		return s.block.NewSub(l, r)
	case lexer.TYPE_ASSIGN:
		val := n.Left.calc(m, f, s)
		s.block.NewStore(r, val)
		return val
	default:
		panic("unexpected op")
	}
}

type NumNode struct {
	Val value.Value
}

func (n *NumNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return n.Val
}

type UnaryNode struct {
	Op    int
	Child Node
}

var zero = constant.NewInt(types.I32, 0)

func (n *UnaryNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	c := loadIfVar(n.Child, m, f, s)
	switch n.Op {
	case lexer.TYPE_PLUS:
		return c
	case lexer.TYPE_SUB:
		return s.block.NewSub(zero, c)
	default:
		panic("unexpected op")
	}
}

type VarNode struct {
	ID []string
}

func (n *VarNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if len(n.ID) == 1 {
		v, err := s.searchVar(n.ID[0])
		if err != nil {
			panic(fmt.Errorf("variable %s not defined", n.ID))
		}
		return v
	}
	va, err := s.searchVar(n.ID[0])
	if err != nil {
		// TODO module
		panic(fmt.Errorf("variable %s not defined", n.ID))
	}

	s1 := reflect.Indirect(reflect.ValueOf(va).Elem()).FieldByName("ElemType").MethodByName("Name").Call([]reflect.Value{})[0].String()
	for _, v := range n.ID[1:] {
		tp := globalScope.getStruct(s1)
		fi := tp.fieldsIdx[v]
		va = s.block.NewGetElementPtr(tp.structType, va,
			constant.NewIndex(zero),
			constant.NewIndex(constant.NewInt(types.I32, int64(fi.idx))))
		s1 = fi.ftype.Name()
	}
	return va
}

// SLNode statement list node
type SLNode struct {
	Children []Node
}

func (n *SLNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	for _, v := range n.Children {
		v.calc(m, f, s)
	}
	return zero
}

type ProgramNode struct {
	Children []Node
}

func (n *ProgramNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	n.Emit(m)
	return zero
}
func (n *ProgramNode) Emit(m *ir.Module) value.Value {

	// define all structs
	for {
		failed := []func(m *ir.Module) error{}
		for _, v := range globalScope.defFuncs {
			if v(m) != nil {
				failed = append(failed, v)
			}
		}
		globalScope.defFuncs = failed
		if len(failed) == 0 {
			break
		}
	}
	for _, v := range n.Children {
		v.calc(m, nil, globalScope)
	}
	return zero
}

type EmptyNode struct {
}

func (n *EmptyNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return zero
}

type DefineNode struct {
	ID  string
	TP  *TypeNode
	Val value.Value
}

func (n *DefineNode) V() value.Value {
	return n.Val
}

func (n *DefineNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if strings.Contains(n.ID, ".") {
		panic("unexpected '.' in varname")
	}
	tp, err := n.TP.calc()
	if err != nil {
		panic(err)
	}
	if f == nil {
		n.Val = m.NewGlobal(n.ID, tp)
	} else {
		n.Val = s.block.NewAlloca(tp)
		s.addVar(n.ID, n.Val)
	}
	return n.Val
}

type ParamNode struct {
	ID  string
	TP  *TypeNode
	Val value.Value
}

func (n *ParamNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if strings.Contains(n.ID, ".") {
		panic("unexpected '.' in paramname")
	}
	tp, err := n.TP.calc()
	if err != nil {
		panic(err)
	}
	n.Val = ir.NewParam(n.ID, tp)
	return n.Val
}
func (n *ParamNode) V() value.Value {
	return n.Val
}

type ParamsNode struct {
	Params []Node
}

func (n *ParamsNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {

	return zero
}

type FuncNode struct {
	Params       Node
	ID           string
	RetType      *TypeNode
	Statements   Node
	Fn           *ir.Func
	DefaultBlock *ir.Block
}

func (n *FuncNode) AddtoScope() {
	if strings.Contains(n.ID, ".") {
		panic("unexpected '.' in funcname")
	}
	psn := n.Params.(*ParamsNode)
	ps := []*ir.Param{}
	for _, v := range psn.Params {
		p := v.(*ParamNode)
		tp, err := n.RetType.calc()
		if err != nil {
			panic(err)
		}
		param := ir.NewParam(p.ID, tp)
		ps = append(ps, param)
	}
	tp, err := n.RetType.calc()
	if err != nil {
		panic(err)
	}
	globalScope.addVar(n.ID, ir.NewFunc(n.ID, tp, ps...))
}

func (n *FuncNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	// _, err := s.searchVar(n.ID)
	// if err == nil {
	// 	panic(fmt.Sprintf("re defination of func %s", n.ID))
	// }
	psn := n.Params.(*ParamsNode)
	ps := []*ir.Param{}
	childScope := s.addChildScope(nil)
	for _, v := range psn.Params {
		param := v.calc(m, f, s).(*ir.Param)
		ps = append(ps, param)
	}
	tp, err := n.RetType.calc()
	if err != nil {
		panic(err)
	}
	fn := m.NewFunc(n.ID, tp, ps...)
	n.Fn = fn
	b := fn.NewBlock("")
	childScope.block = b

	n.DefaultBlock = b
	for i, v := range ps {
		ptr := b.NewAlloca(v.Type())
		b.NewStore(v, ptr)
		childScope.addVar(psn.Params[i].(*ParamNode).ID, ptr)
	}

	s.addVar(n.ID, n.Fn)

	n.Statements.calc(m, fn, childScope)
	return fn
}

type CallFuncNode struct {
	Params []Node
	ID     string
}

func (n *CallFuncNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	params := []value.Value{}
	for _, v := range n.Params {
		params = append(params, loadIfVar(v, m, f, s))
	}
	fn, err := s.searchVar(n.ID)
	if err != nil {
		panic(err)
	}
	return s.block.NewCall(fn, params...)
}

type RetNode struct {
	Exp Node
}

func (n *RetNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	s.block.NewRet(loadIfVar(n.Exp, m, f, s))
	return zero
}

type BoolConstNode struct {
	Val bool
}

func (n *BoolConstNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
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

func (n *CompareNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	l, r := loadIfVar(n.Left, m, f, s), loadIfVar(n.Right, m, f, s)
	hasF := hasFloatType(l, r)
	if hasF {
		return s.block.NewFCmp(comparedic[n.Op].FloatE, l, r)
	} else {
		return s.block.NewICmp(comparedic[n.Op].IntE, l, r)
	}
}

type BoolExpNode struct {
	Op    int
	Left  Node
	Right Node
}

func (n *BoolExpNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	l, r := loadIfVar(n.Left, m, f, s), loadIfVar(n.Right, m, f, s)
	if n.Op == lexer.TYPE_AND {
		return s.block.NewAnd(l, r)
	} else {
		return s.block.NewOr(l, r)
	}
}

type NotNode struct {
	Bool Node
}

func (n *NotNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return s.block.NewICmp(enum.IPredEQ, loadIfVar(n.Bool, m, f, s), constant.False)
}

type IfNode struct {
	BoolExp    Node
	Statements Node
}

var blockID = 100

func (n *IfNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
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

func (n *IfElseNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
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

type DefAndAssignNode struct {
	Val Node
	ID  string
}

func (n *DefAndAssignNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if strings.Contains(n.ID, ".") {
		panic("unexpected '.'")
	}
	if f != nil {
		val := loadIfVar(n.Val, m, f, s)
		v := s.block.NewAlloca(val.Type())
		s.addVar(n.ID, v)
		s.block.NewStore(val, v)
		return v
	}
	// TODO
	panic("not impl")
}

type ForNode struct {
	Bool         Node
	DefineAssign Node
	Assign       Node
	Statements   Node
}

func (n *ForNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
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
		s.block.NewCondBr(loadIfVar(n.Bool, m, f, s), body, end)
	} else {
		s.block.NewBr(body)
	}
	s.block = end
	n.Statements.calc(m, f, child)
	if n.Assign != nil {
		n.Assign.calc(m, f, condScope)
	}
	if n.Bool != nil {
		cond.NewCondBr(loadIfVar(n.Bool, m, f, child), body, end)
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

func (n *BreakNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if s.breakBlock == nil {
		panic("cannot break out of loop")
	}
	s.block.NewBr(s.breakBlock)
	return zero
}

type ContinueNode struct {
}

func (n *ContinueNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if s.continueBlock == nil {
		panic("cannot continue out of loop")
	}
	s.block.NewBr(s.continueBlock)
	return zero
}

type structDefNode struct {
	id     string
	fields map[string]*TypeNode
}

func NewStructDefNode(id string, fieldsMap map[string]*TypeNode) Node {
	n := &structDefNode{id: id, fields: fieldsMap}
	defFunc := func(m *ir.Module) error {
		fields := []types.Type{}
		fieldsIdx := map[string]*field{}
		i := 0
		for k, v := range n.fields {
			tp, err := v.calc()
			if err != nil {
				return err
			}
			fields = append(fields, tp)
			fieldsIdx[k] = &field{
				idx:   i,
				ftype: fields[i],
			}
			i++
		}
		globalScope.addStruct(n.id, &typedef{
			fieldsIdx:  fieldsIdx,
			structType: m.NewTypeDef(n.id, types.NewStruct(fields...)),
		})
		return nil
	}
	globalScope.defFuncs = append(globalScope.defFuncs, defFunc)
	return n

}

func (n *structDefNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return zero
}

type StructInitNode struct {
	ID     []string
	Fields map[string]Node
}

func (n *StructInitNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if len(n.ID) > 1 {
		panic("not impl yet")
	} else {
		tp := globalScope.getStruct(n.ID[0])
		alloca := s.block.NewAlloca(tp.structType)
		for k, v := range n.Fields {
			fi := tp.fieldsIdx[k]
			ptr := s.block.NewGetElementPtr(tp.structType, alloca,
				constant.NewIndex(zero),
				constant.NewIndex(constant.NewInt(types.I32, int64(fi.idx))))
			s.block.NewStore(loadIfVar(v, m, f, s), ptr)
		}
		return alloca
	}
}

type TypeNode struct {
	ResType  int
	CustomTp []string
	Arr      *Array
}

type Array struct {
	Len      int
	InnerArr *Array
}

func (v *TypeNode) calc() (types.Type, error) {
	if len(v.CustomTp) == 0 {
		return typedic[v.ResType], nil
	} else {
		if len(v.CustomTp) == 1 {
			s := globalScope.getStruct(v.CustomTp[0])
			if s == nil {
				return nil, errVarNotFound
			}
			return s.structType, nil
		} else {
			panic("not impl")
		}
	}
}

type ArrayInitNode struct {
	Type *TypeNode
	Vals []Node
}

func (n *ArrayInitNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	panic("not impl")
}
