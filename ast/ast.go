package ast

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
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
		lexer.TYPE_RES_VOID:    types.Void,
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
		// if nd, ok := n.Right.(*VarBlockNode); ok {
		// 	getVarNode(n.Left).setHeap(nd.getHeap(s), s)
		// } else {
		// 	if all, ok := rawR.(*ir.InstAlloca); ok {
		// 		getVarNode(n.Left).setHeap(mallocTable[all], s)
		// 	}
		// 	getVarNode(n.Left).setHeap(false, s)
		// }
		return val
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

func store(r, lptr value.Value, s *scope) value.Value {
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

func (n *NumNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return n.Val
}

type UnaryNode struct {
	Op    int
	Child Node
}

var zero = constant.NewInt(lexer.DefaultIntType(), 0)

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
	Token       string
	Idxs        []Node
	parent      value.Value
	Next        *VarBlockNode
	allocOnHeap bool
}
type alloca interface {
	setAlloc(onheap bool)
}

func (n *VarBlockNode) setAlloc(onheap bool) {
	n.allocOnHeap = onheap
}

func (n *VarBlockNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	var va value.Value
	if n.parent == nil {
		// head node
		var err error
		var val *variable
		val, err = s.searchVar(n.Token)
		va = val.v
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
	if len(idxs) > 0 {
		// dereference the pointer
		va = deReference(va, s)
	}
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
	va = deReference(va, s)
	n.Next.parent = va
	return n.Next.calc(m, f, s)
}

func deReference(va value.Value, s *scope) value.Value {
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

func (n *SLNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	var escMap = map[string][]*escNode{}
	var defMap = map[string]bool{}
	var escPoint = []string{}
	var heapAllocTable = map[string]bool{}
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
			right := getVarNode(node.Val)
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
		case *RetNode:
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
		case *CallFuncNode:
			for _, v := range node.Params {
				right := getVarNode(v)
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
			}

		}
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
		} else {
			v.initNode.setAlloc(true)
		}
	}
}

type ProgramNode struct {
	Children []Node
}

func (n *ProgramNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	n.Emit(m)
	return zero
}
func (n *ProgramNode) Emit(m *ir.Module) value.Value {
	// define all interfaces
	for _, v := range globalScope.interfaceDefFuncs {
		v()
	}

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
	tp, err := n.TP.calc(s)
	if err != nil {
		panic(err)
	}
	if f == nil {
		// TODO global
		if ptr, ok := tp.(*types.PointerType); ok {
			n.Val = m.NewGlobalDef(n.ID, constant.NewNull(ptr))
		} else {
			// TODO
		}
		s.addVar(n.ID, &variable{n.Val})
	} else {
		if s.heapAllocTable[n.ID] {
			gfn := globalScope.genericFuncs["heapalloc"]
			fnv := gfn(m, s, n.TP)
			n.Val = s.block.NewCall(fnv)
			s.addVar(n.ID, &variable{n.Val})
		} else {
			n.Val = s.block.NewAlloca(tp)
			s.addVar(n.ID, &variable{n.Val})
		}
	}
	return n.Val
}

var mallocTable = map[*ir.InstAlloca]bool{}

type RetNode struct {
	Exp Node
}

func (n *RetNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if n.Exp == nil {
		s.block.NewRet(nil)
		return zero
	}
	ret := n.Exp.calc(m, f, s)
	v, err := implicitCast(loadIfVar(ret, s), f.Sig.RetType, s)
	if err != nil {
		panic(err)
	}
	s.block.NewRet(v)
	return zero
}

type NilNode struct {
}

func (n *NilNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	return zero
}

type DefAndAssignNode struct {
	Val Node
	ID  string
}

func autoAlloc(m *ir.Module, id string, gtp TypeNode, tp types.Type, s *scope) (v value.Value) {
	if s.heapAllocTable[id] {
		gfn := globalScope.genericFuncs["heapalloc"]
		fnv := gfn(m, s, gtp)
		v = s.block.NewCall(fnv)
	} else {
		v = s.block.NewAlloca(tp)
	}
	return

}

func (n *DefAndAssignNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	if strings.Contains(n.ID, ".") {
		panic("unexpected '.'")
	}
	if f != nil {
		rawval := n.Val.calc(m, f, s)
		val := loadIfVar(rawval, s)
		var v value.Value
		var heap bool
		var tp types.Type
		switch val.Type().(type) {
		case *types.FloatType:
			tp = lexer.DefaultFloatType()
			v = autoAlloc(m, n.ID,
				&BasicTypeNode{ResType: lexer.TYPE_RES_FLOAT},
				tp, s)

		case *types.IntType:
			if val.Type().(*types.IntType).BitSize == 1 {
				tp = val.Type()
				v = autoAlloc(m, n.ID,
					&BasicTypeNode{ResType: lexer.TYPE_RES_BOOL},
					tp, s)
			} else {
				tp = lexer.DefaultIntType()
				v = autoAlloc(m, n.ID,
					&BasicTypeNode{ResType: lexer.TYPE_RES_INT},
					tp, s)
			}
		default:
			tp = val.Type()
			v = autoAlloc(m, n.ID,
				&calcedTypeNode{tp},
				tp, s)
		}
		var val1 = v
		val, err := implicitCast(val, tp, s)
		if err != nil {
			panic(err)
		}
		va := &variable{v}
		store(val, val1, s)
		if heap {
			s.addVar(n.ID, va)
			return v
		}
		s.addVar(n.ID, va)
		return v
	}
	// TODO
	panic("not impl")
}

func implicitCast(v value.Value, target types.Type, s *scope) (value.Value, error) {
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
	case *types.PointerType:
		v = deReference(v, s)
		tp, ok := target.(*interf)
		src := strings.Trim(v.Type().String(), "%*")
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
