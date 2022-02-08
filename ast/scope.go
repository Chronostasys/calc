package ast

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type scope struct {
	parent            *scope
	vartable          map[string]*variable
	childrenScopes    []*scope
	block             *ir.Block
	continueBlock     *ir.Block
	breakBlock        *ir.Block
	types             map[string]*typedef
	defFuncs          []func(m *ir.Module) error
	interfaceDefFuncs []func()
	funcDefFuncs      []func()
	genericFuncs      map[string]func(m *ir.Module, s *scope, gens ...TypeNode) value.Value
	genericMap        map[string]types.Type
	heapAllocTable    map[string]bool
}

type variable struct {
	v    value.Value
	heap *varheap
}

type varheap struct {
	heap     bool
	innervar map[string]*varheap
}

func (v *varheap) onheap() bool {
	if v.heap {
		return true
	}
	if v.innervar != nil {
		for _, v := range v.innervar {
			if v.onheap() {
				return true
			}
		}
	}
	return false
}

type typedef struct {
	structType types.Type
	fieldsIdx  map[string]*field
	interf     bool
	funcs      map[string]*FuncNode
}

type field struct {
	idx   int
	ftype types.Type
}

func newScope(block *ir.Block) *scope {
	return &scope{
		vartable:     make(map[string]*variable),
		block:        block,
		types:        map[string]*typedef{},
		genericFuncs: make(map[string]func(m *ir.Module, s *scope, gens ...TypeNode) value.Value),
		genericMap:   make(map[string]types.Type),
	}
}

func (s *scope) addChildScope(block *ir.Block) *scope {
	child := newScope(block)
	child.parent = s
	child.continueBlock = s.continueBlock
	child.breakBlock = s.breakBlock
	// child.genericMap = s.genericMap
	s.childrenScopes = append(s.childrenScopes, child)
	return child
}

var errRedef = fmt.Errorf("variable redefination in same scope")

func (s *scope) addVar(id string, val *variable) error {
	_, ok := s.vartable[id]
	if ok {
		return errRedef
	}
	s.vartable[id] = val
	return nil
}
func (s *scope) addGeneric(id string, val func(m *ir.Module, s *scope, gens ...TypeNode) value.Value) error {
	_, ok := s.genericFuncs[id]
	if ok {
		return errRedef
	}
	s.genericFuncs[id] = val
	return nil
}

var errVarNotFound = fmt.Errorf("variable defination not found")

func (s *scope) searchVar(id string) (*variable, error) {
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.vartable[id]
		if ok {
			if val.heap == nil {
				val.heap = &varheap{}
			}
			if s.heapAllocTable[id] {
				val.heap.heap = s.heapAllocTable[id]
			}
			return val, nil
		}
		scope = scope.parent
	}
	return nil, errVarNotFound
}

func (s *scope) addStruct(id string, structT *typedef) error {
	_, ok := s.types[id]
	if ok {
		return errRedef
	}
	s.types[id] = structT
	return nil
}

func (s *scope) getStruct(id string) *typedef {
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

func (s *scope) getGenericType(id string) types.Type {
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
