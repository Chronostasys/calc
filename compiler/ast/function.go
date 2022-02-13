package ast

import (
	"fmt"
	"strings"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type ParamNode struct {
	ID  string
	TP  TypeNode
	Val value.Value
}

func (n *ParamNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	tp, err := n.TP.calc(s)
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
	Params []*ParamNode
	Ext    bool
}

func (n *ParamsNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {

	return zero
}

type FuncNode struct {
	Params       *ParamsNode
	ID           string
	RetType      TypeNode
	Statements   Node
	Fn           *ir.Func
	DefaultBlock *ir.Block
	Generics     []string
}

func (n *FuncNode) AddtoScope(s *Scope) {
	if len(n.Generics) > 0 {
		s.globalScope.addGeneric(n.ID, func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value {
			sig := fmt.Sprintf("%s<", n.ID)
			for i, v := range n.Generics {
				var tp types.Type
				if i >= len(gens) {

					tp = s.genericMap[v]
				} else {
					tp, _ = gens[i].calc(s)
				}
				s.genericMap[v] = tp
				if i != 0 {
					sig += ","
				}
				sig += tp.String()
			}
			sig += ">"
			fn, err := s.globalScope.searchVar(sig)
			if err == nil {
				return fn.v
			}
			psn := n.Params
			ps := []*ir.Param{}
			for _, v := range psn.Params {
				p := v
				tp, err := p.TP.calc(s)
				if err != nil {
					panic(err)
				}
				param := ir.NewParam(p.ID, tp)
				ps = append(ps, param)
			}
			tp, err := n.RetType.calc(s)
			if err != nil {
				panic(err)
			}
			fun := m.NewFunc(s.getFullName(sig), tp, ps...)
			n.Fn = fun
			s.globalScope.addVar(sig, &variable{v: fun})
			b := fun.NewBlock("")
			childScope := s.addChildScope(b)
			n.DefaultBlock = b
			for i, v := range ps {
				ptr := b.NewAlloca(v.Type())
				store(v, ptr, childScope)
				childScope.addVar(psn.Params[i].ID, &variable{v: ptr})
			}
			n.Statements.calc(m, fun, childScope)
			return fun
		})
		return
	} else {
		s.globalScope.funcDefFuncs = append(s.globalScope.funcDefFuncs, func(s *Scope) {
			psn := n.Params
			ps := []*ir.Param{}
			for _, v := range psn.Params {
				p := v
				tp, err := p.TP.calc(s.globalScope)
				if err != nil {
					panic(err)
				}
				param := ir.NewParam(p.ID, tp)
				ps = append(ps, param)
			}
			tp, err := n.RetType.calc(s.globalScope)
			if err != nil {
				panic(err)
			}
			s.globalScope.addVar(n.ID, &variable{v: ir.NewFunc(s.getFullName(n.ID), tp, ps...)})
		})
	}
}

func (n *FuncNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if len(n.Generics) > 0 {
		// generic function will be generate while call
		return zero
	}
	psn := n.Params
	ps := []*ir.Param{}
	childScope := s.addChildScope(nil)
	for _, v := range psn.Params {
		param := v.calc(m, f, s).(*ir.Param)
		ps = append(ps, param)
	}
	tp, err := n.RetType.calc(s)
	if err != nil {
		panic(err)
	}
	fn := m.NewFunc(s.getFullName(n.ID), tp, ps...)
	n.Fn = fn
	b := fn.NewBlock("")
	childScope.block = b

	n.DefaultBlock = b
	for i, v := range ps {
		ptr := b.NewAlloca(v.Type())
		store(v, ptr, childScope)
		childScope.addVar(psn.Params[i].ID, &variable{v: ptr})
	}

	s.addVar(n.ID, &variable{v: n.Fn})

	n.Statements.calc(m, fn, childScope)
	return fn
}

type CallFuncNode struct {
	Params   []Node
	FnNode   Node
	parent   value.Value
	Next     Node
	Generics []TypeNode
}

func (n *CallFuncNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	var fn *ir.Func

	params := []value.Value{}
	pvs := []value.Value{}
	poff := 0
	varNode := n.FnNode.(*VarBlockNode)
	fnNode := varNode
	scope, ok := ScopeMap[varNode.Token]
	if !ok {
		scope = s
		prev := fnNode
		for {
			if fnNode.Next == nil {
				s := prev.Next
				prev.Next = nil
				defer func() {
					prev.Next = s
				}()
				break
			}
			prev = fnNode
			fnNode = fnNode.Next
		}
	}
	if n.parent != nil {
		varNode = nil
		goto EXT
	}

	for _, v := range n.Params {
		v2 := v.calc(m, f, s)
		v1 := loadIfVar(v2, s)
		pvs = append(pvs, v1)
	}
EXT:
	if fnNode != varNode {
		var alloca value.Value
		if varNode != nil {
			alloca = deReference(varNode.calc(m, f, s), s)
		} else {
			alloca = n.parent
		}
		name := strings.Trim(alloca.Type().String(), "%*\"")
		idx := strings.Index(name, "<")
		if idx > -1 {
			name = name[:idx]
		}
		ss := helper.SplitLast(name, ".")
		if len(ss) > 1 && !strings.Contains(ss[1], "/") { // method is in another module
			mod := ss[0]
			scope = ScopeMap[mod]
			for k, v := range s.genericMap {
				scope.genericMap[k] = v
			}
		}
		name = name + "." + fnNode.Token
		var err error
		var fnv value.Value
		if len(n.Generics) > 0 {
			if gfn := scope.getGenericFunc(name); gfn != nil {
				fnv = gfn(m, n.Generics...)
			} else {
				panic(fmt.Errorf("cannot find generic method %s", name))
			}
		} else {
			var va *variable
			va, err = scope.searchVar(name)
			fnv = va.v
			if err != nil {
				panic(err)
			}
		}
		fn = fnv.(*ir.Func)
		if _, ok := fn.Sig.Params[0].(*types.PointerType); ok {
			alloca = deReference(alloca, s)
		} else {
			alloca = loadIfVar(alloca, s)
		}
		params = append(params, alloca)
		poff = 1
	} else {
		if len(n.Generics) > 0 {
			token := fnNode.Token
			if !ok {
				scope = s.globalScope
			} else {
				token = varNode.Next.Token
			}
			if gfn := scope.getGenericFunc(token); gfn != nil {
				fn = gfn(m, n.Generics...).(*ir.Func)
			} else {
				panic(fmt.Errorf("cannot find generic method %s", fnNode.Token))
			}
		} else {
			fn = fnNode.calc(m, f, s).(*ir.Func)
		}
	}
	for i, v := range pvs {
		tp := fn.Params[i+poff].Typ
		v1 := v
		p, err := implicitCast(v1, tp, s)
		if err != nil {
			panic(err)
		}
		params = append(params, p)
	}

	var re value.Value = s.block.NewCall(fn, params...)
	if !re.Type().Equal(types.Void) {
		// autoAlloc()
		alloc := s.block.NewAlloca(re.Type())
		store(re, alloc, s)
		if fnNode.Token == "heapalloc" {
			mallocTable[alloc] = true
		}
		re = alloc
	}
	if n.Next != nil {
		re = deReference(re, s)
		switch next := n.Next.(type) {
		case *CallFuncNode:
			next.parent = re
		case *VarBlockNode:
			next.parent = re
		default:
			panic("unknown type")
		}
		re = n.Next.calc(m, f, s)
	}
	return re
}
