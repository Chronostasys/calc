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
		lexer.TYPE_RES_FLOAT:   lexer.DefaultFloatType(),
		lexer.TYPE_RES_INT:     lexer.DefaultIntType(),
		lexer.TYPE_RES_BOOL:    types.I1,
		lexer.TYPE_RES_FLOAT32: types.Float,
		lexer.TYPE_RES_INT32:   types.I32,
		lexer.TYPE_RES_FLOAT64: types.Double,
		lexer.TYPE_RES_INT64:   types.I64,
		lexer.TYPE_RES_BYTE:    types.I8,
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

func loadIfVar(l value.Value, s *scope) value.Value {

	if t, ok := l.Type().(*types.PointerType); ok {
		return s.block.NewLoad(t.ElemType, l)
	}
	return l
}

func hasFloatType(b *ir.Block, ts ...value.Value) (bool, []value.Value) {
	for _, v := range ts {
		switch v.Type().(type) {
		case *types.FloatType:
		case *types.IntType:
			tp := v.Type().(*types.IntType)
			if tp.BitSize == 1 {
				return false, ts
			}
		default:
			return false, ts
		}
	}
	hasfloat := false
	var maxF *types.FloatType = types.Half
	var maxI *types.IntType = types.I8
	for _, v := range ts {
		t, ok := v.Type().(*types.FloatType)
		if ok {
			hasfloat = true
			if t.Kind > maxF.Kind {
				maxF = t
			}
		} else {
			tp := v.Type().(*types.IntType)
			if tp.BitSize > maxI.BitSize {
				maxI = tp
			}
		}
	}
	re := []value.Value{}
	for _, v := range ts {
		if hasfloat {
			t, ok := v.Type().(*types.FloatType)
			if ok {
				if t.Kind == maxF.Kind {
					re = append(re, v)
				} else {
					re = append(re, b.NewFPExt(v, maxF))
				}
			} else {
				re = append(re, b.NewSIToFP(v, maxF))
			}
		} else {
			t := v.Type().(*types.IntType)
			if t.BitSize == maxI.BitSize {
				re = append(re, v)
			} else {
				re = append(re, b.NewZExt(v, maxI))
			}
		}
	}

	return hasfloat, re
}

func (n *BinNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	rawL, rawR := n.Left.calc(m, f, s), n.Right.calc(m, f, s)
	l, r := loadIfVar(rawL, s), loadIfVar(rawR, s)
	hasF, re := hasFloatType(s.block, l, r)
	l, r = re[0], re[1]
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
		val := rawL
		// if _, ok := n.Right.(*NumNode); ok {

		// }
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
	c := loadIfVar(n.Child.calc(m, f, s), s)
	switch n.Op {
	case lexer.TYPE_PLUS:
		return c
	case lexer.TYPE_SUB:
		hasF, re := hasFloatType(s.block, c)
		if hasF {
			return s.block.NewFSub(constant.NewFloat(c.Type().(*types.FloatType), 0), re[0])
		}
		return s.block.NewSub(zero, c)
	default:
		panic("unexpected op")
	}
}

func getElmType(v interface{}) types.Type {
	return reflect.Indirect(reflect.ValueOf(v).Elem()).FieldByName("ElemType").Interface().(types.Type)
}

func getTypeName(v interface{}) string {
	return reflect.Indirect(reflect.ValueOf(v).Elem()).FieldByName("ElemType").MethodByName("Name").Call([]reflect.Value{})[0].String()
}

type VarBlockNode struct {
	Token  string
	Idxs   []Node
	parent value.Value
	Next   *VarBlockNode
}

func (n *VarBlockNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	var va value.Value
	if n.parent == nil {
		// head node
		var err error
		va, err = s.searchVar(n.Token)
		if err != nil {
			// TODO module
			panic(fmt.Errorf("variable %s not defined", n.Token))
		}
	} else {
		va = n.parent
		s1 := getTypeName(va.Type())
		tp := globalScope.getStruct(s1)
		fi := tp.fieldsIdx[n.Token]
		va = s.block.NewGetElementPtr(tp.structType, va,
			constant.NewIndex(zero),
			constant.NewIndex(constant.NewInt(types.I32, int64(fi.idx))))
	}
	idxs := n.Idxs
	for _, v := range idxs {
		tp := getElmType(va.Type())
		idx := loadIfVar(v.calc(m, f, s), s)
		if _, ok := idx.Type().(*types.IntType); !ok {
			// TODO indexer reload
			panic("not impl")
		}
		va = s.block.NewGetElementPtr(tp, va,
			constant.NewIndex(zero),
			idx,
		)
	}
	if n.Next == nil {
		return va
	}

	// dereference the pointer
	tpptr := va.Type()
	for {
		if ptr, ok := tpptr.(*types.PointerType); ok {
			tpptr = ptr.ElemType
			if _, ok := tpptr.(*types.PointerType); ok {
				va = s.block.NewLoad(tpptr, va)
			} else {
				break
			}
		}
	}
	n.Next.parent = va
	return n.Next.calc(m, f, s)
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
	// add all func declaration to scope
	for _, v := range globalScope.funcDefFuncs {
		v()
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
	TP  TypeNode
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
	TP  TypeNode
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
	RetType      TypeNode
	Statements   Node
	Fn           *ir.Func
	DefaultBlock *ir.Block
}

func (n *FuncNode) AddtoScope() {
	globalScope.funcDefFuncs = append(globalScope.funcDefFuncs, func() {
		if strings.Contains(n.ID, ".") {
			panic("unexpected '.' in funcname")
		}
		psn := n.Params.(*ParamsNode)
		ps := []*ir.Param{}
		for _, v := range psn.Params {
			p := v.(*ParamNode)
			tp, err := p.TP.calc()
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
	})
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
	FnNode Node
}

func (n *CallFuncNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	fn := n.FnNode.calc(m, f, s)
	params := []value.Value{}
	for i, v := range n.Params {
		tp := fn.(*ir.Func).Params[i].Typ
		p, err := implicitCast(loadIfVar(v.calc(m, f, s), s), tp, s)
		if err != nil {
			panic(err)
		}
		params = append(params, p)
	}
	re := s.block.NewCall(fn, params...)
	if re.Type().Equal(types.Void) {
		return re
	}
	alloc := s.block.NewAlloca(re.Type())
	s.block.NewStore(re, alloc)
	return alloc
}

type RetNode struct {
	Exp Node
}

func (n *RetNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	v, err := implicitCast(loadIfVar(n.Exp.calc(m, f, s), s), f.Sig.RetType, s)
	if err != nil {
		panic(err)
	}
	s.block.NewRet(v)
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
	l, r := loadIfVar(n.Left.calc(m, f, s), s), loadIfVar(n.Right.calc(m, f, s), s)
	hasF, re := hasFloatType(s.block, l, r)
	l, r = re[0], re[1]
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

func (n *NotNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return s.block.NewICmp(enum.IPredEQ, loadIfVar(n.Bool.calc(m, f, s), s), constant.False)
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
		val := loadIfVar(n.Val.calc(m, f, s), s)
		var v *ir.InstAlloca
		switch val.Type().(type) {
		case *types.FloatType:
			v = s.block.NewAlloca(lexer.DefaultFloatType())
		case *types.IntType:
			if val.Type().(*types.IntType).BitSize == 1 {
				v = s.block.NewAlloca(val.Type())
			} else {
				v = s.block.NewAlloca(lexer.DefaultIntType())
			}
		default:
			v = s.block.NewAlloca(val.Type())
		}
		val, err := implicitCast(val, v.ElemType, s)
		if err != nil {
			panic(err)
		}
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
	fields map[string]TypeNode
}

func NewStructDefNode(id string, fieldsMap map[string]TypeNode) Node {
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

func implicitCast(v value.Value, target types.Type, s *scope) (value.Value, error) {
	if v.Type().Equal(target) {
		return v, nil
	}
	switch v.Type().(type) {
	case *types.FloatType:
		tp := v.Type().(*types.FloatType)
		targetTp := target.(*types.FloatType)
		if targetTp.Kind < tp.Kind {
			return nil, fmt.Errorf("failed to perform impliciot cast from %T to %v", v, target)
		}
		return s.block.NewFPExt(v, targetTp), nil
	case *types.IntType:
		tp := v.Type().(*types.IntType)
		targetTp := target.(*types.IntType)
		if targetTp.BitSize < tp.BitSize {
			return nil, fmt.Errorf("failed to perform impliciot cast from %T to %v", v, target)
		}
		return s.block.NewZExt(v, targetTp), nil
	default:
		return nil, fmt.Errorf("failed to cast %T to %v", v, target)
	}
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
			va, err := implicitCast(loadIfVar(v.calc(m, f, s), s), fi.ftype, s)
			if err != nil {
				panic(err)
			}
			s.block.NewStore(va, ptr)
		}
		return alloca
	}
}

type BasicTypeNode struct {
	ResType  int
	CustomTp []string
	PtrLevel int
}

type TypeNode interface {
	calc() (types.Type, error)
	SetPtrLevel(int)
}

type ArrayTypeNode struct {
	Len      int
	ElmType  TypeNode
	PtrLevel int
}

func (v *ArrayTypeNode) SetPtrLevel(i int) {
	v.PtrLevel = i
}
func (v *BasicTypeNode) SetPtrLevel(i int) {
	v.PtrLevel = i
}
func (v *ArrayTypeNode) calc() (types.Type, error) {
	elm, err := v.ElmType.calc()
	if err != nil {
		return nil, err
	}
	var tp types.Type
	tp = types.NewArray(uint64(v.Len), elm)
	for i := 0; i < v.PtrLevel; i++ {
		tp = types.NewPointer(tp)
	}
	return tp, nil
}

func (v *BasicTypeNode) calc() (types.Type, error) {
	var s types.Type
	if len(v.CustomTp) == 0 {
		s = typedic[v.ResType]
	} else {
		if len(v.CustomTp) == 1 {
			st := globalScope.getStruct(v.CustomTp[0])
			if st == nil {
				return nil, errVarNotFound
			}
			s = st.structType
		} else {
			panic("not impl")
		}
	}
	if s == nil {
		return nil, errVarNotFound
	}
	for i := 0; i < v.PtrLevel; i++ {
		s = types.NewPointer(s)
	}
	return s, nil
}

type ArrayInitNode struct {
	Type TypeNode
	Vals []Node
}

func (n *ArrayInitNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	tp := n.Type
	atype, err := tp.calc()
	if err != nil {
		panic(err)
	}
	alloca := s.block.NewAlloca(atype)
	for k, v := range n.Vals {
		ptr := s.block.NewGetElementPtr(atype, alloca,
			constant.NewIndex(zero),
			constant.NewIndex(constant.NewInt(types.I32, int64(k))))
		s.block.NewStore(loadIfVar(v.calc(m, f, s), s), ptr)
	}
	return alloca
}

type TakePtrNode struct {
	Node Node
}

func (n *TakePtrNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	v := n.Node.calc(m, f, s)
	ptr := s.block.NewAlloca(v.Type())
	s.block.NewStore(v, ptr)
	return ptr
}

type TakeValNode struct {
	Level int
	Node  Node
}

func (n *TakeValNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	v := n.Node.calc(m, f, s)

	for i := 0; i < n.Level; i++ {
		v = s.block.NewLoad(getElmType(v.Type()), v)
	}
	return v

}
