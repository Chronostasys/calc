package ast

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var (
	typedic = map[int]types.Type{
		lexer.TYPE_RES_FLOAT:   lexer.DefaultFloatType(),
		lexer.TYPE_RES_INT:     lexer.DefaultIntType(),
		lexer.TYPE_RES_BOOL:    types.I1,
		lexer.TYPE_RES_FLOAT32: types.Float,
		lexer.TYPE_RES_INT32:   types.I32,
		lexer.TYPE_RES_FLOAT64: types.Double,
		lexer.TYPE_RES_INT64:   types.I64,
		lexer.TYPE_RES_BYTE:    types.I8,
		lexer.TYPE_RES_VOID:    types.Void,
		lexer.TYPE_RES_STR:     getstrtp(),
	}
	initf = ir.NewFunc("init.params", types.Void)
	initb = initf.NewBlock("")
)

func getstrtp() types.Type {
	s := types.NewStruct()
	s.TypeName = "github.com/Chronostasys/calc/runtime._str"
	return s
}

type Node interface {
	calc(*ir.Module, *ir.Func, *Scope) value.Value
	travel(func(Node))
}

type ExpNode interface {
	Node
	tp() TypeNode
}

type BinNode struct {
	Op    int
	Left  ExpNode
	Right ExpNode
}

func (b *BinNode) tp() TypeNode {
	if _, ok := b.Left.(*NilNode); ok {
		return b.Right.tp()
	}
	return b.Left.tp()
}

func loadIfVar(l value.Value, s *Scope) value.Value {

	if t, ok := l.Type().(*types.PointerType); ok {
		if _, ok := t.ElemType.(*types.FuncType); ok {
			return l
		}
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

func (n *BinNode) travel(f func(Node)) {
	f(n)
	n.Left.travel(f)
	n.Right.travel(f)
}

func (n *BinNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	rawR := n.Right.calc(m, f, s)
	r := loadIfVar(rawR, s)
	if n.Op == lexer.TYPE_ASSIGN {
		s.rightValue = r
	}
	rawL := n.Left.calc(m, f, s)
	s.rightValue = nil
	l := loadIfVar(rawL, s)
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
		if s.assigned {
			s.assigned = false
			return zero
		}
		if _, ok := n.Right.(*NilNode); ok {
			r = constant.NewNull(rawL.Type().(*types.PointerType).ElemType.(*types.PointerType))
			store(r, rawL, s)
			return rawL
		}
		val := rawL
		r, err := implicitCast(r, l.Type(), s)
		if err != nil {
			panic(err)
		}
		store(r, val, s)
		return zero
	case lexer.TYPE_PS:
		if hasF {
			return s.block.NewFRem(l, r)
		}
		return s.block.NewSRem(l, r)
	case lexer.TYPE_SHL:
		return s.block.NewShl(l, r)
	case lexer.TYPE_SHR:
		return s.block.NewAShr(l, r)
	case lexer.TYPE_BIT_OR:
		return s.block.NewOr(l, r)
	case lexer.TYPE_BIT_XOR:
		return s.block.NewXor(l, r)
	case lexer.TYPE_ESP:
		return s.block.NewAnd(l, r)
	default:
		panic("unexpected op")
	}
}

func getVarNode(n Node) alloca {

	for {
		if node, ok := n.(*TakeValNode); ok {
			n = node.Node
		} else if node, ok := n.(*TakePtrNode); ok {
			n = node.Node
		} else {
			a, _ := n.(alloca)
			return a
		}
	}
}

func store(r, lptr value.Value, s *Scope) value.Value {
	if r.Type().Equal(lptr.Type().(*types.PointerType).ElemType) {
		s.block.NewStore(r, lptr)
		return lptr
	}
	if _, ok := lptr.Type().(*types.PointerType).ElemType.(*interf); ok {
		store := &ir.InstStore{Src: r, Dst: lptr}
		s.block.Insts = append(s.block.Insts, store)
		return lptr
	}

	panic("store failed")
}

type NumNode struct {
	Val value.Value
}

func (n *NumNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return n.Val
}
func (n *NumNode) travel(f func(Node)) {
	f(n)
}
func (n *NumNode) tp() TypeNode {
	return &calcedTypeNode{n.Val.Type()}
}

type UnaryNode struct {
	Op    int
	Child ExpNode
}

func (n *UnaryNode) tp() TypeNode {
	return n.Child.tp()
}
func (n *UnaryNode) travel(f func(Node)) {
	f(n)
	n.Child.travel(f)
}

var zero = constant.NewInt(types.I32, 0)

func (n *UnaryNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
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

func getTypeName(v types.Type) string {
	return strings.Trim(v.String(), "%*\"")
}

type VarBlockNode struct {
	Token       string
	Idxs        []Node
	parent      value.Value
	Next        *VarBlockNode
	allocOnHeap bool
}

var (
	tpm = ir.NewModule()
	tpf = tpm.NewFunc("tmp", types.Void)
	tps = newScope(tpf.NewBlock(""))
)

func (b *VarBlockNode) tp() TypeNode {
	return &calcedTypeNode{b.calc(tpm, tpf, tps).Type()}
}

func (n *VarBlockNode) travel(f func(Node)) {
	f(n)
}

type alloca interface {
	setAlloc(onheap bool)
}

func (n *VarBlockNode) setAlloc(onheap bool) {
	n.allocOnHeap = onheap
}

func (n *VarBlockNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	var va value.Value
	if n.parent == nil {
		// head node
		var err error
		var val *variable
		val, err = s.searchVar(n.Token)
		if err != nil {
			scope := ScopeMap[n.Token]
			val, err = scope.searchVar(n.Next.Token)
			if err != nil {
				panic(fmt.Errorf("variable %s not defined", n.Token))
			}
			n = n.Next
		}
		va = val.v
	} else {
		va = n.parent

		s1 := getTypeName(va.Type())
		s2 := helper.SplitLast(s1, ".")
		ss := s2[0]
		var scope = s
		if len(s2) > 1 {
			ss = s2[1]
			scope = ScopeMap[s2[0]]
		}
		tp := scope.getStruct(ss)
		fi := tp.fieldsIdx[n.Token]
		va = s.block.NewGetElementPtr(tp.structType, va,
			constant.NewIndex(zero),
			constant.NewIndex(constant.NewInt(types.I32, int64(fi.idx))))
	}
	idxs := n.Idxs
	if len(idxs) > 0 {
		// dereference the pointer
		va = deReference(va, s)
	}
	for _, v := range idxs {
		innerTP := va.Type().(*types.PointerType).ElemType
		if atp, ok := innerTP.(*types.ArrayType); ok {
			tp := atp
			idx := loadIfVar(v.calc(m, f, s), s)
			va = s.block.NewGetElementPtr(tp, va,
				constant.NewIndex(zero),
				idx,
			)
		} else {
			va = n.getReloadIdx(va, idxs, m, f, s)
		}
	}
	if n.Next == nil {
		return va
	}

	// dereference the pointer
	va = deReference(va, s)
	n.Next.parent = va
	return n.Next.calc(m, f, s)
}

type fakeNode struct {
	v value.Value
}

func (n *fakeNode) travel(f func(Node)) {
	f(n)
}

func (n *fakeNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return n.v
}

func (n *VarBlockNode) getReloadIdx(val value.Value, idxs []Node, m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.Next == nil && s.rightValue != nil {
		i := INDEX_RELOAD

		for iter, v := range idxs {
			ps := []Node{v}
			if len(idxs)-1 == iter {
				i = INDEX_SET_RELOAD
				ps = append(ps, &fakeNode{s.rightValue})
			}
			b := &VarBlockNode{Token: i}
			cf := &CallFuncNode{
				FnNode: b,
				Params: ps,
				parent: val,
			}
			val = cf.calc(m, f, s)
		}
		s.assigned = true
		return val
	}
	b := &VarBlockNode{Token: INDEX_RELOAD}
	for _, v := range idxs {
		cf := &CallFuncNode{
			FnNode: b,
			Params: []Node{v},
			parent: val,
		}
		val = cf.calc(m, f, s)
	}
	return val
}

func deReference(va value.Value, s *Scope) value.Value {
	tpptr := va.Type()
	for {
		if ptr, ok := tpptr.(*types.PointerType); ok {
			tpptr = ptr.ElemType
			if _, ok := tpptr.(*types.PointerType); ok {
				va = s.block.NewLoad(tpptr, va)
			} else {
				if inter, ok := tpptr.(*interf); ok {
					// interface type, return it's real type
					realTP := inter.innerType

					return s.block.NewIntToPtr(s.block.NewLoad(tpptr, va), types.NewPointer(realTP))
				}
				break
			}
		}
	}
	return va
}

// SLNode statement list node
type SLNode struct {
	Children []Node
}

type escNode struct {
	token    string
	initNode alloca
}

func (n *SLNode) travel(f func(Node)) {
	f(n)
	for _, v := range n.Children {
		v.travel(f)
	}
}

func (n *SLNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	var escMap = map[string][]*escNode{}
	var defMap = map[string]bool{}
	var escPoint = []string{}
	var heapAllocTable = map[string]bool{}
	var closuredef = map[string]bool{}
	var closurevar = map[string]bool{}
	var trf func(n Node)
	travel := func(fn *InlineFuncNode) {
		fn.Body.travel(trf)
		fn.closureVars = closurevar

	}
	trf = func(n Node) { // // 逃逸点4：闭包
		// 在一个闭包中的逃逸点是闭包引用的外界变量
		switch node := n.(type) {
		case *DefineNode:
			closuredef[node.ID] = true
		case *DefAndAssignNode:
			closuredef[node.ID] = true
		case *VarBlockNode: // 只有varblock第一段的变量构成逃逸
			if !closuredef[node.Token] {
				heapAllocTable[node.Token] = true
				closurevar[node.Token] = true
			}
		case *InlineFuncNode:
			travel(node)

		}
	}
	if strings.Contains(f.Ident(), "heapalloc") ||
		strings.Contains(f.Ident(), "heapfree") ||
		strings.Contains(f.Ident(), "MallocList") {
		goto LOOP
	}
	// stackescape analysis 逃逸分析
	for _, v := range n.Children {
		switch node := v.(type) {
		case *BinNode:
			if node.Op == lexer.TYPE_ASSIGN {
				if r, ok := node.Right.(*InlineFuncNode); ok {
					closurevar = map[string]bool{}
					travel(r)
				}
				left := getVarNode(node.Left).(*VarBlockNode)
				right := getVarNode(node.Right)
				if right == nil {
					continue
				}
				name := left.Token
				if !defMap[left.Token] {
					name = "extern.." + name
				}
				if r, ok := right.(*VarBlockNode); ok {
					rname := r.Token
					if !defMap[r.Token] {
						rname = "extern.." + rname
					}
					escMap[name] = append(escMap[name], &escNode{token: rname})
				} else {
					escMap[name] = append(escMap[name], &escNode{initNode: right})
				}
			}
		case *DefAndAssignNode:
			defMap[node.ID] = true
			name := node.ID
			if r, ok := node.ValNode.(*InlineFuncNode); ok {
				travel(r)
			}
			right := getVarNode(node.ValNode)
			if right == nil {
				continue
			}
			if r, ok := right.(*VarBlockNode); ok {
				rname := r.Token
				if !defMap[r.Token] {
					rname = "extern.." + rname
				}
				escMap[name] = append(escMap[name], &escNode{token: rname})
			} else {
				escMap[name] = append(escMap[name], &escNode{initNode: right})
			}
		case *DefineNode:
			defMap[node.ID] = true
		case *RetNode: // 逃逸点1：返回值
			if r, ok := node.Exp.(*InlineFuncNode); ok {
				travel(r)
			}
			right := getVarNode(node.Exp)
			if right == nil {
				continue
			}
			if r, ok := right.(*VarBlockNode); ok {
				rname := r.Token
				if !defMap[r.Token] {
					rname = "extern.." + rname
				}
				escPoint = append(escPoint, rname)
			} else {
				right.setAlloc(true)
			}
		case *CallFuncNode: // 逃逸点2：方法参数
			for _, v := range node.Params {
				right := getVarNode(v)
				if right == nil {
					continue
				}
				// TODO TakeValNode & TakePtrNode? 这不确定有没有问题
				if r, ok := right.(*VarBlockNode); ok {
					rname := r.Token
					if !defMap[r.Token] {
						rname = "extern.." + rname
					}
					escPoint = append(escPoint, rname)
				} else {
					right.setAlloc(true)
				}
			}
		}
	}
	for _, v := range f.Params { // 逃逸点3：给入参赋值
		escPoint = append(escPoint, "extern.."+v.LocalName)
	}
	for _, v := range escPoint {
		if defMap[v] {
			heapAllocTable[v] = true
		}
		next := escMap[v]
		delete(escMap, v)

		findEsc(next, defMap, heapAllocTable, escMap)
	}
	s.heapAllocTable = heapAllocTable
LOOP:
	for _, v := range n.Children {
		v.calc(m, f, s)
	}
	return zero
}
func findEsc(next []*escNode, defMap map[string]bool, heapAllocTable map[string]bool, escMap map[string][]*escNode) {
	if next == nil {
		return
	}
	for _, v := range next {
		if v.initNode == nil && defMap[v.token] {
			heapAllocTable[v.token] = true
			next = escMap[v.token]
			delete(escMap, v.token)
			findEsc(next, defMap, heapAllocTable, escMap)
		} else if v.initNode != nil {
			v.initNode.setAlloc(true)
		}
	}
}

type ProgramNode struct {
	PKG         *PackageNode
	Imports     *ImportNode
	Children    []Node
	GlobalScope *Scope
}

func (n *ProgramNode) travel(f func(Node)) {
	f(n)
	for _, v := range n.Children {
		v.travel(f)
	}
}

func Merge(ns ...*ProgramNode) *ProgramNode {
	ss := []*Scope{}
	p := &ProgramNode{}
	for _, v := range ns {
		ss = append(ss, v.GlobalScope)
		p.Children = append(p.Children, v.Children...)
		if p.PKG != nil && p.PKG.Name != v.PKG.Name {
			panic("found two different modules under same dir")
		}
		p.PKG = v.PKG
	}
	s := MergeGlobalScopes(ss...)
	p.GlobalScope = s
	return p
}

func (n *ProgramNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	n.Emit(m)
	return zero
}

func (n *ProgramNode) CalcGlobals(m *ir.Module) {

	globalScope := n.GlobalScope
	// define all interfaces
	for _, v := range globalScope.interfaceDefFuncs {
		v(n.GlobalScope)
	}
	globalScope.interfaceDefFuncs = globalScope.interfaceDefFuncs[:0]

	// define all structs
	for {
		failed := []func(m *ir.Module, s *Scope) error{}
		for _, v := range globalScope.defFuncs {
			if v(m, n.GlobalScope) != nil {
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
		v(n.GlobalScope)
	}
	globalScope.funcDefFuncs = globalScope.funcDefFuncs[:0]
	// add all global variables to scope
	for _, v := range n.Children {
		switch v.(type) {
		case *DefineNode, *DefAndAssignNode:
			v.calc(m, nil, globalScope)
		}
	}
}

func (n *ProgramNode) Emit(m *ir.Module) {
	globalScope := n.GlobalScope
	n.CalcGlobals(m)
	for _, v := range n.Children {
		switch v.(type) {
		case *DefineNode, *DefAndAssignNode:
		default:
			v.calc(m, nil, globalScope)
		}
	}
	mi, err := globalScope.searchVar("main")
	if err != nil {
		return
	}
	main := mi.v.(*ir.Func)
	initb.NewRet(nil)
	m.Funcs = append(m.Funcs, initf)
	main.Blocks[0].Insts = append([]ir.Instruction{
		ir.NewCall(initf), // add global init
	}, main.Blocks[0].Insts...)

}

type EmptyNode struct {
}

func (n *EmptyNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}
func (n *EmptyNode) travel(f func(Node)) {
	f(n)
}

type defNode interface {
	Node
	setVal(func(s *Scope) value.Value)
	getID() string
}

type DefineNode struct {
	ID  string
	TP  TypeNode
	Val value.Value
	vf  func(s *Scope) value.Value
}

func (n *DefineNode) setVal(f func(s *Scope) value.Value) {
	n.vf = f
}

func (n *DefineNode) travel(f func(Node)) {
	f(n)
}
func (n *DefineNode) getID() string {
	return n.ID
}

func (n *DefineNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.vf != nil {
		v := n.vf(s)
		s.addVar(n.ID, &variable{v: v})
		return v
	}
	tp, err := n.TP.calc(s)
	if err != nil {
		panic(err)
	}
	if f == nil {
		n.Val = m.NewGlobalDef(s.getFullName(n.ID), constant.NewZeroInitializer(tp))
		s.addVar(n.ID, &variable{v: n.Val})
	} else {
		if s.heapAllocTable[n.ID] {
			n.Val = heapAlloc(m, s, n.TP)
			s.addVar(n.ID, &variable{v: n.Val})
		} else {
			n.Val = stackAlloc(m, s, tp)
			s.addVar(n.ID, &variable{v: n.Val})
		}
	}
	return n.Val
}

var mallocTable = map[*ir.InstAlloca]bool{}

type RetNode struct {
	Exp Node
}

func (n *RetNode) travel(f func(Node)) {
	f(n)
	if n.Exp == nil {
		return
	}
	n.Exp.travel(f)
}

func (n *RetNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.Exp == nil {
		s.block.NewRet(nil)
		return zero
	}
	ret := n.Exp.calc(m, f, s)
	v, err := implicitCast(loadIfVar(ret, s), f.Sig.RetType, s)
	if err != nil {
		panic(err)
	}
	if s.freeFunc != nil {
		s.freeFunc(s)
	}
	s.block.NewRet(v)
	return zero
}

type NilNode struct {
}

func (n *NilNode) tp() TypeNode {
	return nil
}

func (n *NilNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}

func (n *NilNode) travel(f func(Node)) {
	f(n)
}

type DefAndAssignNode struct {
	ValNode Node
	ID      string
	Val     func(s *Scope) value.Value
}

func (n *DefAndAssignNode) getID() string {
	return n.ID
}

func (n *DefAndAssignNode) setVal(v func(s *Scope) value.Value) {
	n.Val = v
}

func (n *DefAndAssignNode) travel(f func(Node)) {
	f(n)
	n.ValNode.travel(f)
}

func autoAlloc(m *ir.Module, id string, gtp TypeNode, tp types.Type, s *Scope) (v value.Value) {
	if s.heapAllocTable[id] {
		v = heapAlloc(m, s, gtp)
	} else {
		v = stackAlloc(m, s, tp)
	}
	return

}

func (n *DefAndAssignNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.Val != nil {
		v := n.Val(s)
		rawval := n.ValNode.calc(m, f, s)
		val := loadIfVar(rawval, s)
		val, err := implicitCast(val, getElmType(v.Type()), s)
		if err != nil {
			panic(err)
		}
		store(val, v, s)
		s.addVar(n.ID, &variable{v: v})
		return v
	}
	global := false
	if f == nil {
		global = true
		f = initf
		f.Parent = m
		s.block = initb
		defer func() {
			s.block = nil
		}()
	}
	rawval := n.ValNode.calc(m, f, s)
	val := loadIfVar(rawval, s)
	var v value.Value
	var tp types.Type
	var tpNode TypeNode
	switch val.Type().(type) {
	case *types.FloatType:
		tp = lexer.DefaultFloatType()
		tpNode = &BasicTypeNode{ResType: lexer.TYPE_RES_FLOAT}
	case *types.IntType:
		if val.Type().(*types.IntType).BitSize == 1 {
			tp = val.Type()
			tpNode = &BasicTypeNode{ResType: lexer.TYPE_RES_BOOL}
		} else {
			tp = lexer.DefaultIntType()
			tpNode = &BasicTypeNode{ResType: lexer.TYPE_RES_INT}
		}
	default:
		tp = val.Type()
		tpNode = &calcedTypeNode{tp}
	}
	if !global {
		v = autoAlloc(m, n.ID,
			tpNode,
			tp, s)
	} else {
		v = m.NewGlobalDef(s.getFullName(n.ID), constant.NewZeroInitializer(tp))
	}

	val, err := implicitCast(val, tp, s)
	if err != nil {
		panic(err)
	}
	va := &variable{v: v}
	store(val, v, s)
	s.addVar(n.ID, va)
	return v
}

func implicitCast(v value.Value, target types.Type, s *Scope) (value.Value, error) {
	if v.Type().Equal(target) {
		return v, nil
	}
	if t, ok := target.(*interf); ok {
		if v.Type().Equal(t.IntType) {
			return v, nil
		}
	}
	switch v.Type().(type) {
	case *types.FloatType:
		tp := v.Type().(*types.FloatType)
		targetTp := target.(*types.FloatType)
		if targetTp.Kind < tp.Kind {
			return nil, fmt.Errorf("failed to perform implicit cast from %T to %v", v, target)
		}
		return s.block.NewFPExt(v, targetTp), nil
	case *types.IntType:
		tp := v.Type().(*types.IntType)
		targetTp := target.(*types.IntType)
		if targetTp.BitSize < tp.BitSize {
			return nil, fmt.Errorf("failed to perform implicit cast from %T to %v", v, target)
		}
		return s.block.NewZExt(v, targetTp), nil
	case *types.PointerType:
		v = deReference(v, s)
		tp, ok := target.(*interf)
		src := strings.Trim(v.Type().String(), "%*\"")
		if ok { // turn to interface
			for k, v1 := range tp.interfaceFuncs {
				fnv, err := s.searchVar(src + "." + k)
				if err != nil {
					goto FAIL
				}
				fn := fnv.v.(*ir.Func)
				for i, u := range v1.Params.Params {
					ptp, err := u.TP.calc(s)
					if err != nil {
						goto FAIL
					}
					if !fn.Sig.Params[i+1].Equal(ptp) {
						goto FAIL
					}
				}
				rtp, err := v1.RetType.calc(s)
				if err != nil {
					goto FAIL
				}
				if !fn.Sig.RetType.Equal(rtp) {
					goto FAIL
				}
			}
			// cast
			inst := s.block.NewPtrToInt(v, lexer.DefaultIntType())
			tp.innerType = v.Type().(*types.PointerType).ElemType
			return inst, nil
		}
	FAIL:
		return nil, fmt.Errorf("failed to cast %v to interface %v", v, target.Name())
	default:
		return nil, fmt.Errorf("failed to cast %v to %v", v, target)
	}
}
