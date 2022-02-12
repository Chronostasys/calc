package ast

import (
	"strings"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type BasicTypeNode struct {
	ResType  int
	CustomTp []string
	PtrLevel int
	Generics []TypeNode
	Pkg      string
}

type TypeNode interface {
	calc(*Scope) (types.Type, error)
	SetPtrLevel(int)
	String(*Scope) string
}

type calcedTypeNode struct {
	tp types.Type
}

func (v *calcedTypeNode) SetPtrLevel(i int) {
	panic("not impl")
}
func (v *calcedTypeNode) calc(*Scope) (types.Type, error) {
	return v.tp, nil
}
func (v *calcedTypeNode) String(*Scope) string {
	panic("not impl")
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
func (v *ArrayTypeNode) String(s *Scope) string {
	t, err := v.calc(s.globalScope)
	if err != nil {
		panic(err)
	}
	tp := strings.Trim(t.String(), "%*\"")
	return tp
}
func (v *BasicTypeNode) String(s *Scope) string {
	m := ir.NewModule()
	oldm := s.m
	s.m = m
	t, err := v.calc(s.globalScope)
	if err != nil {
		panic(err)
	}
	s.m = oldm
	tp := strings.Trim(t.String(), "%*\"")
	return tp
}
func (v *ArrayTypeNode) calc(s *Scope) (types.Type, error) {
	elm, err := v.ElmType.calc(s)
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

func loadElmType(tp types.Type) types.Type {
	for p, ok := tp.(*types.PointerType); ok; p, ok = tp.(*types.PointerType) {
		tp = p.ElemType
	}
	return tp
}

func (v *BasicTypeNode) calc(sc *Scope) (types.Type, error) {
	var s types.Type
	if len(v.CustomTp) == 0 {
		s = typedic[v.ResType]
	} else {
		tpname := v.CustomTp[0]
		getTp := func() {
			if len(v.Generics) > 0 {
				gfn := sc.getGenericStruct(tpname)
				td := gfn(sc.m, v.Generics...)
				s = td.structType
				return
			}
			st := types.NewStruct()
			def := sc.getStruct(tpname)

			if def != nil {
				s = def.structType
			} else if sc.getGenericType(tpname) != nil {
				s = sc.getGenericType(tpname)
			} else {
				st.TypeName = v.Pkg + "." + tpname
				s = st
			}
		}
		if len(v.CustomTp) == 1 {
			getTp()
		} else {
			sc = ScopeMap[v.CustomTp[0]]
			if sc == nil {
				println()
			}
			tpname = v.CustomTp[1]
			getTp()
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
	Type        TypeNode
	Vals        []Node
	allocOnHeap bool
}

func (n *ArrayInitNode) setAlloc(onheap bool) {
	n.allocOnHeap = onheap
}

func (n *ArrayInitNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	tp := n.Type
	atype, err := tp.calc(s)
	if err != nil {
		panic(err)
	}
	var alloca value.Value
	if n.allocOnHeap {
		gfn := s.globalScope.getGenericFunc("heapalloc")
		fnv := gfn(m, n.Type)
		alloca = s.block.NewCall(fnv)
	} else {
		alloca = s.block.NewAlloca(atype)
	}
	var va value.Value = alloca
	for k, v := range n.Vals {
		ptr := s.block.NewGetElementPtr(atype, va,
			constant.NewIndex(zero),
			constant.NewIndex(constant.NewInt(types.I32, int64(k))))
		cs, err := implicitCast(loadIfVar(v.calc(m, f, s), s), atype, s)
		if err != nil {
			panic(err)
		}
		store(cs, ptr, s)
	}
	return alloca
}

func (n *StructInitNode) setAlloc(onheap bool) {
	n.allocOnHeap = onheap
}
func (n *StructInitNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	t, err := n.TP.calc(s)
	if err != nil {
		panic(err)
	}
	ss1 := t.String()
	ss2 := strings.Trim(ss1, "%*\"")
	scs := helper.SplitLast(ss2, ".")
	var scope = s
	var ss string
	if len(scs) > 1 {
		ss = scs[1]
		scope = ScopeMap[scs[0]]
	} else {
		ss = scs[0]
	}
	tp := scope.getStruct(ss)
	if tp == nil {
		panic("failed to find type declareation")
	}
	var alloca value.Value
	if n.allocOnHeap {
		gfn := s.globalScope.getGenericFunc("heapalloc")
		fnv := gfn(m, n.TP)
		alloca = s.block.NewCall(fnv)
	} else {
		alloca = s.block.NewAlloca(tp.structType)
	}

	var va value.Value = alloca
	// assign
	for k, v := range n.Fields {
		fi := tp.fieldsIdx[k]
		ptr := s.block.NewGetElementPtr(tp.structType, va,
			constant.NewIndex(zero),
			constant.NewIndex(constant.NewInt(types.I32, int64(fi.idx))))
		va, err := implicitCast(loadIfVar(v.calc(m, f, s), s), fi.ftype, s)
		if err != nil {
			panic(err)
		}
		store(va, ptr, s)
	}
	return alloca
}

type StructDefNode struct {
	ptrlevel int
	Fields   map[string]TypeNode
	fields   map[string]*field
}

func (v *StructDefNode) SetPtrLevel(i int) {
	v.ptrlevel = i
}
func (v *StructDefNode) calc(s *Scope) (types.Type, error) {
	fields := []types.Type{}
	fieldsIdx := map[string]*field{}
	i := 0
	for k, v := range v.Fields {
		tp, err := v.calc(s)
		if err != nil {
			return nil, err
		}
		fields = append(fields, tp)
		fieldsIdx[k] = &field{
			idx:   i,
			ftype: fields[i],
		}
		i++
	}
	var tp types.Type
	tp = types.NewStruct(fields...)
	v.fields = fieldsIdx
	for i := 0; i < v.ptrlevel; i++ {
		tp = types.NewPointer(tp)
	}

	tmpID := strings.Trim(tp.String(), "%*\"")
	s.types[tmpID] = &typedef{
		structType: tp,
		fieldsIdx:  v.fields,
	}
	return tp, nil
}
func (v *StructDefNode) String(*Scope) string {
	panic("not impl")
}

type interf struct {
	*types.IntType
	interfaceFuncs map[string]*FuncNode
	innerType      types.Type
}

type InterfaceDefNode struct {
	ptrlevel int
	Funcs    map[string]*FuncNode
}

func (v *InterfaceDefNode) SetPtrLevel(i int) {
	v.ptrlevel = i
}
func (v *InterfaceDefNode) calc(s *Scope) (types.Type, error) {
	var tp types.Type
	tp = &interf{
		IntType:        lexer.DefaultIntType(),
		interfaceFuncs: v.Funcs,
	}

	for i := 0; i < v.ptrlevel; i++ {
		tp = types.NewPointer(tp)
	}
	return tp, nil
}
func (v *InterfaceDefNode) String(*Scope) string {
	panic("not impl")
}

type StructInitNode struct {
	TP          TypeNode
	Fields      map[string]Node
	allocOnHeap bool
}

type typeDefNode struct {
	id       string
	tp       types.Type
	generics []string
}

func NewTypeDef(id string, tp TypeNode, generics []string, m *ir.Module, s *Scope) Node {
	if len(generics) == 0 {
		t, err := tp.calc(s)
		if err != nil {
			panic(err)
		}
		var fidx map[string]*field
		if n, ok := tp.(*StructDefNode); ok {
			fidx = n.fields
		}
		n := &typeDefNode{id: id, tp: t, generics: generics}
		defFunc := func(s *Scope) {
			s.globalScope.addStruct(n.id, &typedef{
				structType: m.NewTypeDef(s.getFullName(n.id), t),
				fieldsIdx:  fidx,
			})
		}
		s.globalScope.interfaceDefFuncs = append(s.globalScope.interfaceDefFuncs, defFunc)
		return n
	}
	deffunc := func(m *ir.Module, s *Scope, gens ...TypeNode) *typedef {
		sig := id + "<"
		genericMap := s.genericMap
		if len(gens) > 0 {
			if genericMap == nil {
				genericMap = make(map[string]types.Type)
			}
			for i, v := range gens {
				tp, err := v.calc(s)
				if err != nil {
					panic(err)
				}
				genericMap[generics[i]] = tp
				sig += tp.String() + ","
			}
		}
		sig += ">"
		if td := s.globalScope.getStruct(sig); td != nil {
			return td
		}
		s.genericMap = genericMap
		t, err := tp.calc(s)
		if err != nil {
			panic(err)
		}
		var fidx map[string]*field
		if n, ok := tp.(*StructDefNode); ok {
			fidx = n.fields
		}
		td := &typedef{
			structType: m.NewTypeDef(s.getFullName(sig), t),
			fieldsIdx:  fidx,
		}
		s.globalScope.addStruct(sig, td)
		return td
	}
	s.addGenericStruct(id, deffunc)
	return &typeDefNode{id: id, generics: generics}
}

func (n *typeDefNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}
