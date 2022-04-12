package ast

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var errn = 0

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
	s.TypeName = "github.com/Chronostasys/calc/runtime/strings._str"
	return s
}

type Node interface {
	calc(*ir.Module, *ir.Func, *Scope) value.Value
	travel(func(Node) bool)
}

type ErrSTNode struct {
	File string
	Line int
	Src  string
}

func (n *ErrSTNode) calc(*ir.Module, *ir.Func, *Scope) value.Value {
	fmt.Printf("\033[31m[error]\033[0m: failed to parse statement \n%s\n at line %d. (%s:%d)\n", n.Src, n.Line, n.File, n.Line)
	errn++
	return nil
}
func (n *ErrSTNode) travel(func(Node) bool) {
}

func CheckErr() {
	if errn > 0 {
		log.Fatalf("compile failed with %d errors.", errn)
	}
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
	if l == nilval {
		return l
	}
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

func (n *BinNode) travel(f func(Node) bool) {
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
		r1, err := implicitCast(r, val.Type().(*types.PointerType).ElemType, s)
		if err != nil {
			panic(err)
		}
		store(r1, val, s)
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
	elmtp := lptr.Type().(*types.PointerType).ElemType
	rtp := r.Type()
	if rtp.Equal(elmtp) {
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
func (n *NumNode) travel(f func(Node) bool) {
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
func (n *UnaryNode) travel(f func(Node) bool) {
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
	Pos         int
	Lexer       *lexer.Lexer
	SrcFile     string
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

func (n *VarBlockNode) travel(f func(Node) bool) {
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
			if scope == nil {
				ln, off := n.Lexer.Currpos(n.Pos)
				panic(fmt.Errorf("\033[31m[error]\033[0m: variable %s not defined (%s:%d:%d)", n.Token, n.SrcFile, ln, off))
			}
			val, err = scope.searchVar(n.Next.Token)
			if err != nil {
				ln, off := n.Lexer.Currpos(n.Pos)
				panic(fmt.Errorf("\033[31m[error]\033[0m: variable %s not defined (%s:%d:%d)", n.Token, n.SrcFile, ln, off))
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
	f func(m *ir.Module, f *ir.Func, s *Scope) value.Value
}

func (n *fakeNode) tp() TypeNode {
	panic("not impl")
}

func (n *fakeNode) travel(f func(Node) bool) {
	f(n)
}

func (n *fakeNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.f != nil {
		return n.f(m, f, s)
	}
	return n.v
}

func (n *VarBlockNode) getReloadIdx(val value.Value, idxs []Node, m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.Next == nil && s.rightValue != nil {
		i := INDEX_RELOAD

		for iter, v := range idxs {
			ps := []Node{v}
			if len(idxs)-1 == iter {
				i = INDEX_SET_RELOAD
				ps = append(ps, &fakeNode{v: s.rightValue})
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

func (n *SLNode) travel(f func(Node) bool) {
	f(n)
	for _, v := range n.Children {
		v.travel(f)
	}
}

func (n *SLNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	var escMap = map[string][]*escNode{} // key：变量名   val：给它赋值过的右值
	var defMap = map[string]bool{}
	var escPoint = map[string]bool{}
	var heapAllocTable = map[string]bool{}
	var closuredef = map[string]bool{}
	var closurevar = map[string]bool{}
	var trf func(n Node) bool
	travel := func(fn *InlineFuncNode) {
		old := closurevar
		closurevar = map[string]bool{}
		fn.Body.travel(trf)
		fn.closureVars = closurevar
		for k, v := range closurevar {
			if !old[k] {
				old[k] = v
			}
		}
		closurevar = old

	}
	trf = func(n Node) bool { // // 逃逸点4：闭包
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
		return true
	}

	callfanf := func(node *CallFuncNode) {
		for _, v := range node.Params {
			if in, ok := v.(*InlineFuncNode); ok {
				travel(in)
			}
			v.travel(func(right Node) bool {
				if r, ok := right.(*VarBlockNode); ok {
					rname := r.Token
					if !defMap[r.Token] {
						rname = "extern.." + rname
					}
					escPoint[rname] = true
				} else if r, ok := right.(alloca); ok {
					r.setAlloc(true)
				}
				return true
			})
			node.FnNode.(*VarBlockNode).setAlloc(true)
			escPoint[node.FnNode.(*VarBlockNode).Token] = true
		}
		// println()
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
					travel(r)
				}
				left := getVarNode(node.Left).(*VarBlockNode)
				right := getVarNode(node.Right)
				if right == nil {
					if cn, ok := node.Right.(*CallFuncNode); ok {
						callfanf(cn)
					}
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
					node.Right.travel(func(n Node) bool {
						switch n := n.(type) {
						case *VarBlockNode:
							rname := n.Token
							if !defMap[n.Token] {
								rname = "extern.." + rname
							}
							escMap[name] = append(escMap[name], &escNode{token: rname})
						}
						return true
					})
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
				if cn, ok := node.ValNode.(*CallFuncNode); ok {
					callfanf(cn)
				}
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
				node.ValNode.travel(func(n Node) bool {
					switch n := n.(type) {
					case *VarBlockNode:
						rname := n.Token
						if !defMap[n.Token] {
							rname = "extern.." + rname
						}
						escMap[name] = append(escMap[name], &escNode{token: rname})
					}
					return true
				})
			}
		case *DefineNode:
			defMap[node.ID] = true
		case *RetNode: // 逃逸点1：返回值
			if r, ok := node.Exp.(*InlineFuncNode); ok {
				travel(r)
			}
			right := getVarNode(node.Exp)
			if right == nil {
				if cn, ok := node.Exp.(*CallFuncNode); ok {
					callfanf(cn)
				}
				continue
			}
			if r, ok := right.(*VarBlockNode); ok {
				rname := r.Token
				if !defMap[r.Token] {
					rname = "extern.." + rname
				}
				escPoint[rname] = true
			} else {
				right.setAlloc(true)
				node.Exp.travel(func(n Node) bool {
					switch n := n.(type) {
					case *VarBlockNode:
						rname := n.Token
						if !defMap[n.Token] {
							rname = "extern.." + rname
						}
						escPoint[rname] = true
					}
					return true
				})
			}
		case *CallFuncNode: // 逃逸点2：方法参数
			callfanf(node)
		}
	}
	for _, v := range f.Params { // 逃逸点3：给入参赋值
		escPoint["extern.."+v.LocalName] = true
	}
	for v := range escPoint {
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
		f := func() {
			defer func() {
				err := recover()
				if err != nil {
					fmt.Println(err)
					errn++
				}
			}()
			v.calc(m, f, s)
		}
		f()
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

func (n *ProgramNode) travel(f func(Node) bool) {
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

	globalScope.types = map[string]*typedef{}
	// define all structs & interfaces
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
	realmain := m.NewFunc("main", types.I32)
	entry := realmain.NewBlock("")
	// initgc
	setexe, _ := ScopeMap[RUNTIME].searchVar("GC_set_pages_executable")
	entry.NewCall(setexe.v, constant.NewInt(types.I32, 1))
	setfin, _ := ScopeMap[RUNTIME].searchVar("GC_set_java_finalization")
	entry.NewCall(setfin.v, constant.NewInt(types.I32, 1))
	ini, _ := ScopeMap[RUNTIME].searchVar("GC_init")
	entry.NewCall(ini.v)
	// add global init
	entry.NewCall(initf)
	if ScopeMap[CORO_MOD] != nil {
		fe, _ := ScopeMap[CORO_MOD].searchVar("Exec")
		entry.NewCall(fe.v) // start system threads
	}
	if ScopeMap[LIBUV] != nil {
		fe, _ := ScopeMap[LIBUV].searchVar("StartUVLoop")
		entry.NewCall(fe.v) // start event loop
	}
	ret := entry.NewCall(main)
	if asyncMain {
		fe, _ := ScopeMap[CORO_MOD].searchVar("Exec")
		i := ScopeMap[CORO_SM_MOD].getStruct("StateMachine").structType
		in, err := implicitCast(ret, i, &Scope{block: entry})
		if err != nil {
			panic(err)
		}
		entry.NewCall(fe.v, in) // queue main func
	}
	entry.NewRet(zero)

}

type EmptyNode struct {
}

func (n *EmptyNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}
func (n *EmptyNode) travel(f func(Node) bool) {
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

func (n *DefineNode) travel(f func(Node) bool) {
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
			n.Val = gcmalloc(m, s, n.TP)
			s.addVar(n.ID, &variable{v: n.Val})
		} else {
			n.Val = stackAlloc(m, s, tp)
			s.addVar(n.ID, &variable{v: n.Val})
		}
	}
	return n.Val
}

type RetNode struct {
	Exp   Node
	async bool
}

func (n *RetNode) travel(f func(Node) bool) {
	f(n)
	if n.Exp == nil {
		return
	}
	n.Exp.travel(f)
}

var i = 0

func (n *RetNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.Exp == nil {
		if n.async {
			i++
			nb := f.NewBlock(".exit" + strconv.Itoa(i))
			store(constant.NewBlockAddress(f, nb), s.yieldBlock, s)

			i := ScopeMap[CORO_SM_MOD].getStruct("StateMachine").structType
			sm, _ := implicitCast(f.Params[0], i, s)
			// idx := i.(*interf).interfaceFuncs["GetMutex"].i
			// mu := s.block.NewGetElementPtr(i, sm, zero, constant.NewInt(types.I32, int64(idx)))

			qt, err := ScopeMap[CORO_MOD].searchVar("TryQueueContinuous")

			if err != nil {
				panic(err)
			}

			fqt := qt.v.(*ir.Func)
			s.block.NewCall(fqt, sm)

			s.block.NewRet(constant.False)
			s.block = nb
			s.block.NewRet(constant.False)
		} else {
			s.block.NewRet(nil)
		}

		return zero
	}
	ret := n.Exp.calc(m, f, s)
	l := loadIfVar(ret, s)
	rtp := f.Sig.RetType
	if n.async {
		rtp = getElmType(s.yieldRet.Type())
	}
	v, err := implicitCast(l, rtp, s)
	if err != nil {
		panic(err)
	}
	if s.freeFunc != nil {
		s.freeFunc(s)
	}
	if n.async {
		i++
		nb := f.NewBlock(".exit" + strconv.Itoa(i))
		store(constant.NewBlockAddress(f, nb), s.yieldBlock, s)
		store(v, s.yieldRet, s)

		qt, _ := ScopeMap[CORO_MOD].searchVar("TryQueueContinuous")

		fqt := qt.v.(*ir.Func)
		i := ScopeMap[CORO_SM_MOD].getStruct("StateMachine").structType
		sm, err := implicitCast(f.Params[0], i, s)
		if err != nil {
			panic(err)
		}
		s.block.NewCall(fqt, sm)

		s.block.NewRet(constant.False)
		s.block = nb
		s.block.NewRet(constant.False)
		return zero
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
	return nilval
}

func (n *NilNode) travel(f func(Node) bool) {
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

func (n *DefAndAssignNode) travel(f func(Node) bool) {
	f(n)
	n.ValNode.travel(f)
}

func autoAlloc(m *ir.Module, id string, gtp TypeNode, tp types.Type, s *Scope) (v value.Value) {
	if s.heapAllocTable[id] {
		v = gcmalloc(m, s, gtp)
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
	if _, ok := n.ValNode.(*CallFuncNode); ok {
		tp = val.Type()
		tpNode = &calcedTypeNode{tp}
	} else {
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
	}
	if !global {
		v = autoAlloc(m, n.ID,
			tpNode,
			tp, s)
	} else {
		v = m.NewGlobalDef(s.getFullName(n.ID), constant.NewZeroInitializer(tp))
	}

	val1, err := implicitCast(val, tp, s)
	if err != nil {
		panic(err)
	}
	va := &variable{v: v}
	store(val1, v, s)
	s.addVar(n.ID, va)
	return v
}

var nilval = constant.NewNull(types.I8Ptr)

func implicitCast(v value.Value, target types.Type, s *Scope) (value.Value, error) {
	if v == nilval {
		return constant.NewNull(target.(*types.PointerType)), nil
	}
	if v.Type().Equal(target) {
		return v, nil
	}
	switch val := v.Type().(type) {
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
		if tp, ok := target.(*types.PointerType); ok { // handle function type cast
			if val.ElemType.Equal(tp.ElemType) {
				return v, nil
			}
			return nil, fmt.Errorf("failed to cast %v to interface %v", v, target.Name())
		}
		tp, ok := target.(*interf)
		src := strings.Trim(v.Type().String(), "%*\"")
		idx := strings.Index(src, "<")
		if idx > -1 {
			s.generics = nil
			ss := strings.ReplaceAll(src[idx+1:len(src)-1], "\\22", "")
			src = src[:idx]
			lev := 0
			start := 0
			end := 0
			for i, v := range ss {
				if v == '<' {
					lev++
				}
				if v == '>' {
					lev--
				}
				if v == ',' && lev == 0 {
					end = i
					t := types.NewStruct()
					idx = strings.Index(ss[start:end], "*")
					t.TypeName = strings.Trim(ss[start:end], "*%")
					var tp types.Type = t
					if idx > -1 {
						for i := 0; i < end-start-idx; i++ {
							tp = types.NewPointer(tp)
						}
					}
					s.generics = append(s.generics, tp)
					start = end
				}

			}
		}
		if ok { // turn to interface
			st := stackAlloc(s.m, s, tp)
			for k, v1 := range tp.interfaceFuncs {
				f := s.block.NewGetElementPtr(tp.Type, st, zero, constant.NewInt(types.I32, int64(v1.i)))
				// old := s.genericMap
				// s.genericMap = tp.genericMaps
				st := strings.Split(src, ".")[0]
				la := strings.LastIndex(src, "/")
				if la > -1 {
					i := strings.Index(src[la:], ".")
					st = src[:la] + src[la:la+i]
				}
				scope, ok := ScopeMap[st]
				if !ok || scope.Pkgname == s.Pkgname {
					scope = s
				}
				scope.generics = s.generics
				fnv, err := scope.searchVar(src + "." + k)
				// s.genericMap = old
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
				in := s.block.NewPtrToInt(fn, lexer.DefaultIntType())
				store(in, f, s)

			}
			// cast
			inst := s.block.NewPtrToInt(v, lexer.DefaultIntType())
			ptr := s.block.NewGetElementPtr(tp.Type, st, zero, zero)
			store(inst, ptr, s)
			return loadIfVar(st, s), nil
		}
	FAIL:
		return nil, fmt.Errorf("failed to cast %v to interface %v", v, target.Name())
	case *interf:
		tp, ok := target.(*interf)
		if ok {
			st := stackAlloc(s.m, s, tp)
			val2 := stackAlloc(s.m, s, val)
			store(v, val2, s)
			for k, v1 := range tp.interfaceFuncs {
				f := s.block.NewGetElementPtr(tp.Type, st, zero, constant.NewInt(types.I32, int64(v1.i)))
				v2, ok := val.interfaceFuncs[k]
				if !ok {
					goto FAIL1
				}
				for i, p := range v2.Params.Params {
					tp1, err := p.TP.calc(s)
					if err != nil {
						return nil, err
					}
					tp2, err := v1.Params.Params[i].TP.calc(s)
					if err != nil {
						return nil, err
					}
					if !tp1.Equal(tp2) {
						goto FAIL1
					}
				}
				rtp1, err := v1.RetType.calc(s)
				if err != nil {
					goto FAIL1
				}
				rtp2, err := v2.RetType.calc(s)
				if err != nil {
					goto FAIL1
				}
				if !rtp1.Equal(rtp2) {
					goto FAIL1
				}
				f2 := s.block.NewGetElementPtr(val.Type, val2, zero, constant.NewInt(types.I32, int64(v2.i)))
				loadf := loadIfVar(f2, s)
				store(loadf, f, s)
			}
			// cast
			inst := s.block.NewGetElementPtr(val.Type, val2, zero, zero)
			ptr := s.block.NewGetElementPtr(tp.Type, st, zero, zero)
			store(loadIfVar(inst, s), ptr, s)
			return loadIfVar(st, s), nil
		}
	FAIL1:
		return nil, fmt.Errorf("failed to cast %v to interface %v", v, target.Name())
	case *types.ArrayType:
		v1 := gcmalloc(s.m, s, &calcedTypeNode{val})
		store(v, v1, s)
		head := s.block.NewGetElementPtr(val, v1, zero, zero)
		gfn := ScopeMap[SLICE].getGenericFunc("FromArr")
		slicef := gfn(s.m, &calcedTypeNode{val.ElemType})
		slice := s.block.NewCall(slicef, head, constant.NewInt(types.I32, int64(val.Len)))
		if target.Equal(slice.Type()) {
			return slice, nil
		}
		return nil, fmt.Errorf("failed to cast %v to %v", v, target.Name())
	default:
		return nil, fmt.Errorf("failed to cast %v to %v", v, target)
	}
}
