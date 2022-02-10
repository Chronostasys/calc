package ast

import (
	"fmt"
	"strings"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type BasicTypeNode struct {
	ResType  int
	CustomTp []string
	PtrLevel int
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
	tp := strings.Trim(t.String(), "%*")
	return tp
}
func (v *BasicTypeNode) String(s *Scope) string {
	t, err := v.calc(s.globalScope)
	if err != nil {
		panic(err)
	}
	tp := strings.Trim(t.String(), "%*")
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

func (v *BasicTypeNode) calc(sc *Scope) (types.Type, error) {
	var s types.Type
	if len(v.CustomTp) == 0 {
		s = typedic[v.ResType]
	} else {
		tpname := v.CustomTp[0]
		getTp := func() {
			st := types.NewStruct()
			def := sc.getStruct(tpname)
			if def != nil && def.interf {
				s = &interf{
					IntType:        lexer.DefaultIntType(),
					interfaceFuncs: def.funcs,
					name:           tpname,
				}

			} else if sc.getGenericType(tpname) != nil {
				s = sc.getGenericType(tpname)
			} else {
				st.TypeName = sc.getFullName(tpname)
				s = st
			}
		}
		if len(v.CustomTp) == 1 {
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
	var tp *typedef
	if len(n.ID) > 1 {
		scope, ok := ScopeMap[n.ID[0]]
		if !ok {
			panic(fmt.Sprintf("module %s not found", n.ID[0]))
		}
		tp = scope.getStruct(n.ID[1])
	} else {
		tp = s.globalScope.getStruct(n.ID[0])
	}
	var alloca value.Value
	if n.allocOnHeap {
		gfn := s.globalScope.getGenericFunc("heapalloc")
		fnv := gfn(m, &BasicTypeNode{CustomTp: n.ID})
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

type structDefNode struct {
	id     string
	fields map[string]TypeNode
}

func NewStructDefNode(id string, fieldsMap map[string]TypeNode, s *Scope) Node {
	n := &structDefNode{id: id, fields: fieldsMap}
	defFunc := func(m *ir.Module, s *Scope) error {
		fields := []types.Type{}
		fieldsIdx := map[string]*field{}
		i := 0
		for k, v := range n.fields {
			tp, err := v.calc(s.globalScope)
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
		s.globalScope.addStruct(n.id, &typedef{
			fieldsIdx:  fieldsIdx,
			structType: m.NewTypeDef(s.getFullName(n.id), types.NewStruct(fields...)),
		})
		return nil
	}
	s.globalScope.defFuncs = append(s.globalScope.defFuncs, defFunc)
	return n

}

func (n *structDefNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	return zero
}

type StructInitNode struct {
	ID          []string
	Fields      map[string]Node
	allocOnHeap bool
}
