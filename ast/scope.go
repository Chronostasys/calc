package ast

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/value"
)

type scope struct {
	parent         *scope
	vartable       map[string]value.Value
	childrenScopes []*scope
	block          *ir.Block
	continueBlock  *ir.Block
	breakBlock     *ir.Block
}

func newScope(block *ir.Block) *scope {
	return &scope{
		vartable: make(map[string]value.Value),
		block:    block,
	}
}

func (s *scope) addChildScope(block *ir.Block) *scope {
	child := newScope(block)
	child.parent = s
	child.continueBlock = s.continueBlock
	child.breakBlock = s.breakBlock
	s.childrenScopes = append(s.childrenScopes, child)
	return child
}

var errRedef = fmt.Errorf("variable redefination in same scope")

func (s *scope) addVar(id string, val value.Value) error {
	_, ok := s.vartable[id]
	if ok {
		return errRedef
	}
	s.vartable[id] = val
	return nil
}

var errVarNotFound = fmt.Errorf("variable defination not found")

func (s *scope) searchVar(id string) (value.Value, error) {
	scope := s
	for {
		if scope == nil {
			break
		}
		val, ok := scope.vartable[id]
		if ok {
			return val, nil
		}
		scope = scope.parent
	}
	return nil, errVarNotFound
}
