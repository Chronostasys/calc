package ast

import (
	"fmt"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type Scope struct {
	Pkgname           string
	globalScope       *Scope
	parent            *Scope
	vartable          map[string]*variable
	childrenScopes    []*Scope
	block             *ir.Block
	continueBlock     *ir.Block
	breakBlock        *ir.Block
	types             map[string]*typedef
	defFuncs          []func(m *ir.Module, s *Scope) error
	interfaceDefFuncs []func(s *Scope)
	funcDefFuncs      []func(s *Scope)
	genericFuncs      map[string]func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value
	genericStructs    map[string]func(m *ir.Module, s *Scope, gens ...TypeNode) *typedef
	genericMap        map[string]types.Type
	heapAllocTable    map[string]bool
	m                 *ir.Module
	generics          []types.Type
	paramGenerics     [][]types.Type
	currParam         int
	rightValue        value.Value
	assigned          bool
}

var externMap = map[string]bool{
	"printf":    true,
	"memset":    true,
	"GC_malloc": true,
	"memcpy":    true,
	"Sleep":     true,
}

func MergeGlobalScopes(ss ...*Scope) *Scope {
	s := NewGlobalScope(ss[0].m)
	s.Pkgname = ss[0].Pkgname
	for _, v := range ss {
		for id, v := range v.vartable {
			s.addVar(id, v)
		}
		for k, v := range v.types {
			s.addStruct(k, v)
		}
		for k, v := range v.genericFuncs {
			s.addGeneric(k, v)
		}
		for k, v := range v.genericStructs {
			s.addGenericStruct(k, v)
		}

		s.defFuncs = append(s.defFuncs, v.defFuncs...)
		s.interfaceDefFuncs = append(s.interfaceDefFuncs, v.interfaceDefFuncs...)
		s.funcDefFuncs = append(s.funcDefFuncs, v.funcDefFuncs...)
		s.childrenScopes = append(s.childrenScopes, v.childrenScopes...)
		for _, v := range v.childrenScopes {
			v.parent = s
		}
	}
	return s
}

type variable struct {
	v        value.Value
	generics []types.Type
}

type typedef struct {
	structType types.Type
	fieldsIdx  map[string]*field
	generics   []types.Type
}

type field struct {
	idx   int
	ftype types.Type
}

func newScope(block *ir.Block) *Scope {
	sc := &Scope{
		vartable:       make(map[string]*variable),
		block:          block,
		types:          map[string]*typedef{},
		genericFuncs:   make(map[string]func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value),
		genericMap:     make(map[string]types.Type),
		genericStructs: make(map[string]func(m *ir.Module, s *Scope, gens ...TypeNode) *typedef),
	}
	return sc
}
func NewGlobalScope(m *ir.Module) *Scope {
	sc := newScope(nil)
	sc.globalScope = sc
	sc.m = m
	return sc
}

func (s *Scope) addChildScope(block *ir.Block) *Scope {
	child := newScope(block)
	child.parent = s
	child.continueBlock = s.continueBlock
	child.breakBlock = s.breakBlock
	// child.genericMap = s.genericMap
	child.globalScope = s.globalScope
	child.Pkgname = s.Pkgname
	child.m = s.m
	s.childrenScopes = append(s.childrenScopes, child)
	return child
}

var errRedef = fmt.Errorf("variable redefination in same scope")

func (s *Scope) addVar(id string, val *variable) error {
	id = s.getFullName(id)
	_, ok := s.vartable[id]
	if ok {
		return errRedef
	}
	val.generics = s.generics
	s.vartable[id] = val
	return nil
}
func (s *Scope) getFullName(id string) string {
	if id == "main" {
		return id
	}
	if id[0] == '{' { // anonymous struct
		return id
	}
	if externMap[id] {
		return id
	}

	if strings.Index(id, s.Pkgname+".") != 0 {
		id = s.Pkgname + "." + id
		return id
	}
	return id
}
func (s *Scope) addGeneric(id string, val func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value) error {
	id = s.getFullName(id)
	_, ok := s.genericFuncs[id]
	if ok {
		return errRedef
	}
	s.genericFuncs[id] = val
	return nil
}
func (s *Scope) addGenericStruct(id string, val func(m *ir.Module, s *Scope, gens ...TypeNode) *typedef) error {
	id = s.getFullName(id)
	_, ok := s.genericStructs[id]
	if ok {
		return errRedef
	}
	s.genericStructs[id] = val
	return nil
}

func (s *Scope) getGenericStruct(id string) func(m *ir.Module, gens ...TypeNode) *typedef {
	id = s.getFullName(id)
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.genericStructs[id]
		if ok {
			return func(m *ir.Module, gens ...TypeNode) *typedef {
				return val(m, s, gens...)
			}
		}
		scope = scope.parent
	}
	return nil
}

var errVarNotFound = fmt.Errorf("variable defination not found")

func (s *Scope) searchVar(id string) (*variable, error) {
	id = s.getFullName(id)
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.vartable[id]
		if ok {
			s.generics = val.generics
			return val, nil
		}
		scope = scope.parent
	}
	f := s.getGenericFunc(id)
	if f != nil {
		v := f(s.m)
		if v != nil {
			return &variable{v: v}, nil
		}
	}
	return nil, errVarNotFound
}

func (s *Scope) addStruct(id string, structT *typedef) error {
	id = s.getFullName(id)
	_, ok := s.types[id]
	if ok {
		return errRedef
	}
	s.types[id] = structT
	return nil
}

func (s *Scope) getStruct(id string) *typedef {
	id = s.getFullName(id)
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.types[id]
		if ok {
			return val
		}
		scope = scope.parent
	}
	return nil
}

func (s *Scope) getGenericType(id string) types.Type {
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.genericMap[id]
		if ok {
			return val
		}
		scope = scope.parent
	}
	return nil
}

func (s *Scope) getGenericFunc(id string) func(m *ir.Module, gens ...TypeNode) value.Value {
	id = s.getFullName(id)
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.genericFuncs[id]
		if ok {
			return func(m *ir.Module, gens ...TypeNode) value.Value {
				return val(m, s, gens...)
			}
		}
		scope = scope.parent
	}
	return nil
}

var ScopeMap = map[string]*Scope{}
