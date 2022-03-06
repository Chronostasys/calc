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

func (n *BasicTypeNode) GetPtrLevel() int {
	return n.PtrLevel
}
func (n *BasicTypeNode) Clone() TypeNode {
	return &BasicTypeNode{
		n.ResType, n.CustomTp, n.PtrLevel, n.Generics, n.Pkg,
	}
}

type TypeNode interface {
	calc(*Scope) (types.Type, error)
	SetPtrLevel(int)
	GetPtrLevel() int
	String(*Scope) string
	Clone() TypeNode
}

type FuncTypeNode struct {
	Args     *ParamsNode
	Ret      TypeNode
	ptrlevel int
}

func (n *FuncTypeNode) Clone() TypeNode {
	return &FuncTypeNode{
		n.Args, n.Ret, n.ptrlevel,
	}
}

func (n *FuncTypeNode) GetPtrLevel() int {
	return n.ptrlevel
}

func (v *FuncTypeNode) SetPtrLevel(i int) {
	v.ptrlevel = i
}
func (v *FuncTypeNode) calc(s *Scope) (types.Type, error) {
	var ret types.Type
	if v.Ret != nil {
		r, err := v.Ret.calc(s)
		if err != nil {
			return nil, err
		}
		ret = r
	}
	args := []types.Type{}
	for _, v := range v.Args.Params {
		arg, err := v.TP.calc(s)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	var fn types.Type
	fn = types.NewFunc(ret, args...)
	v.ptrlevel++
	for i := 0; i < v.ptrlevel; i++ {
		fn = types.NewPointer(fn)
	}
	v.ptrlevel--
	return fn, nil
}
func (v *FuncTypeNode) String(*Scope) string {
	panic("not impl")
}

type calcedTypeNode struct {
	tp types.Type
}

func (n *calcedTypeNode) Clone() TypeNode {
	panic("not impl")
}

func (n *calcedTypeNode) GetPtrLevel() int {
	panic("not impl")
}

func (v *calcedTypeNode) SetPtrLevel(i int) {
	panic("not impl")
}
func (v *calcedTypeNode) calc(*Scope) (types.Type, error) {
	return v.tp, nil
}
func (v *calcedTypeNode) String(s *Scope) string {
	t, err := v.calc(s.globalScope)
	if err != nil {
		panic(err)
	}
	tp := strings.Trim(t.String(), "%*\"")
	return tp
}

type ArrayTypeNode struct {
	Len      int
	ElmType  TypeNode
	PtrLevel int
}

func (n *ArrayTypeNode) Clone() TypeNode {
	return &ArrayTypeNode{
		n.Len, n.ElmType, n.PtrLevel,
	}
}

func (n *ArrayTypeNode) GetPtrLevel() int {
	return n.PtrLevel
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
	oris := sc
	if len(v.CustomTp) == 0 {
		s = typedic[v.ResType]
	} else {
		tpname := v.CustomTp[0]
		getTp := func() {
			if len(v.Generics) > 0 {
				gfn := sc.getGenericStruct(tpname)
				if oris.paramGenerics != nil {
					if oris.currParam < len(oris.paramGenerics) {
						gs := oris.paramGenerics[oris.currParam]
						if gs != nil {
							for i, v := range v.Generics {
								if len(v.(*BasicTypeNode).CustomTp) == 0 {
									break
								}
								k := v.(*BasicTypeNode).CustomTp[0]
								ss := strings.Split(k, ".")
								k = ss[len(ss)-1]
								if i < len(gs) {
									oris.genericMap[k] = gs[i]
								}
							}
						}
					}
				}
				td := gfn(sc.m, v.Generics...)
				s = td.structType
				oris.generics = td.generics
				// for k, v := range sc.genericMap {
				// 	oris.genericMap[k] = v
				// }
				return
			}
			st := types.NewStruct()
			def := sc.getStruct(tpname)
			sc.generics = nil
			if def != nil {
				oris.generics = def.generics
				s = def.structType
			} else if sc.getGenericType(tpname) != nil {
				s = sc.getGenericType(tpname)
			} else {
				st.TypeName = v.Pkg + "." + tpname
				s = st
			}
		}
		if len(v.CustomTp) == 1 {
			if sc.Pkgname != v.Pkg {
				sc = ScopeMap[v.Pkg]
			}
			for k, v := range oris.genericMap {
				sc.genericMap[k] = v
			}
			getTp()
		} else {
			sc = ScopeMap[v.CustomTp[0]]
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

func (n *ArrayInitNode) tp() TypeNode {
	return n.Type
}

func (n *ArrayInitNode) setAlloc(onheap bool) {
	n.allocOnHeap = onheap
}

func (n *ArrayInitNode) travel(f func(Node)) {
	f(n)
}

func (n *ArrayInitNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	tp := n.Type
	atype, err := tp.calc(s)
	if err != nil {
		panic(err)
	}
	var alloca value.Value
	if n.allocOnHeap {
		alloca = gcmalloc(m, s, n.Type)
	} else {
		alloca = stackAlloc(m, s, atype)
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
	generics := s.generics
	defer func() {
		s.generics = generics
	}()
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
		alloca = gcmalloc(m, s, n.TP)
	} else {
		alloca = stackAlloc(m, s, tp.structType)
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

	fields        map[string]*field
	Orderedfields []*Field
}

type Field struct {
	Name string
	TP   TypeNode
}

func (n *StructDefNode) Clone() TypeNode {
	return &StructDefNode{
		n.ptrlevel, n.fields, n.Orderedfields,
	}
}

func (n *StructDefNode) GetPtrLevel() int {
	return n.ptrlevel
}

func (v *StructDefNode) SetPtrLevel(i int) {
	v.ptrlevel = i
}
func (v *StructDefNode) calc(s *Scope) (types.Type, error) {
	fields := []types.Type{}
	fieldsIdx := map[string]*field{}
	for i := range v.Orderedfields {
		k, v := v.Orderedfields[i].Name, v.Orderedfields[i].TP
		tp, err := v.calc(s)
		if err != nil {
			return nil, err
		}
		fields = append(fields, tp)
		fieldsIdx[k] = &field{
			idx:   i,
			ftype: fields[i],
		}
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
	types.Type
	interfaceFuncs map[string]*FuncNode
	genericMaps    map[string]types.Type
	id             string
}

func (t *interf) Equal(t1 types.Type) bool {
	if i, ok := t1.(*interf); ok {
		return i.id == t.id
	}
	return false

}

type InterfaceDefNode struct {
	ptrlevel   int
	Funcs      map[string]*FuncNode
	OrderedIDS []string
}

func (n *InterfaceDefNode) Clone() TypeNode {
	return &InterfaceDefNode{
		n.ptrlevel, n.Funcs, n.OrderedIDS,
	}
}

func (n *InterfaceDefNode) GetPtrLevel() int {
	return n.ptrlevel
}

func (v *InterfaceDefNode) SetPtrLevel(i int) {
	v.ptrlevel = i
}
func (v *InterfaceDefNode) calc(s *Scope) (types.Type, error) {
	var tp types.Type
	tps := []types.Type{lexer.DefaultIntType()}
	i := 1
	for _, k := range v.OrderedIDS {
		tps = append(tps, lexer.DefaultIntType())
		v.Funcs[k].i = i
		i++
	}
	interfaceTp := types.NewStruct(tps...)
	tp = &interf{
		Type:           interfaceTp,
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

func (b *StructInitNode) tp() TypeNode {
	return b.TP
}

func (n *StructInitNode) travel(f func(Node)) {
	f(n)
	if n.Fields != nil {
		for _, v := range n.Fields {
			v.travel(f)
		}
	}
}

type typeDefNode struct {
	id       string
	tp       types.Type
	generics []string
}

func (n *typeDefNode) travel(f func(Node)) {
	f(n)
}

func NewTypeDef(id string, tp TypeNode, generics []string, m *ir.Module, s *Scope) Node {
	if len(generics) == 0 {
		sout := s
		n := &typeDefNode{id: id, generics: generics}

		defFunc := func(s *Scope) {
			t, err := tp.calc(sout)
			if err != nil {
				panic(err)
			}
			if tt, ok := t.(*interf); ok {
				tt.id = s.getFullName(n.id)
			}
			var fidx map[string]*field
			if n, ok := tp.(*StructDefNode); ok {
				fidx = n.fields
			}
			n.tp = t

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
		generictypes := []types.Type{}
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
				generictypes = append(generictypes, tp)
				sig += tp.String() + ","
			}
		}
		sig += ">"
		if td := s.globalScope.getStruct(sig); td != nil {
			return td
		}

		// 提前定义好类型占位符，这样才能允许自引用
		tmpss := types.NewStruct()
		tmpss.SetName(s.getFullName(sig))

		td := &typedef{
			structType: tmpss,
			generics:   generictypes,
		}
		s.globalScope.addStruct(sig, td)
		s.genericMap = genericMap
		t, err := tp.calc(s)
		if tt, ok := t.(*interf); ok {
			tt.genericMaps = make(map[string]types.Type)
			for k, v := range genericMap {
				tt.genericMaps[k] = v
			}
			tt.id = s.getFullName(sig)
		}
		if err != nil {
			panic(err)
		}
		var fidx map[string]*field
		if n, ok := tp.(*StructDefNode); ok {
			fidx = n.fields
		}
		td.structType = m.NewTypeDef(s.getFullName(sig), t)
		td.fieldsIdx = fidx
		// td := &typedef{
		// 	structType: m.NewTypeDef(s.getFullName(sig), t),
		// 	fieldsIdx:  fidx,
		// 	generics:   generictypes,
		// }
		// s.globalScope.addStruct(sig, td)
		return td
	}
	s.addGenericStruct(id, deffunc)
	return &typeDefNode{id: id, generics: generics}
}

func (n *typeDefNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}
