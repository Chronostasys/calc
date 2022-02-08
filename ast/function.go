package ast

import (
	"fmt"
	"strings"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type ParamNode struct {
	ID  string
	TP  TypeNode
	Val value.Value
}

func (n *ParamNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
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

func (n *ParamsNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {

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

func (n *FuncNode) AddtoScope() {
	if len(n.Generics) > 0 {
		globalScope.addGeneric(n.ID, func(m *ir.Module, s *scope, gens ...TypeNode) value.Value {
			sig := fmt.Sprintf("%s<", n.ID)
			for i, v := range n.Generics {
				tp, _ := gens[i].calc(s)
				s.genericMap[v] = tp
				if i != 0 {
					sig += ","
				}
				sig += tp.String()
			}
			sig += ">"
			fn, err := globalScope.searchVar(sig)
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
			fun := m.NewFunc(sig, tp, ps...)
			n.Fn = fun
			globalScope.addVar(sig, &variable{fun})
			b := fun.NewBlock("")
			childScope := s.addChildScope(b)
			n.DefaultBlock = b
			for i, v := range ps {
				ptr := b.NewAlloca(v.Type())
				store(v, ptr, childScope)
				childScope.addVar(psn.Params[i].ID, &variable{ptr})
			}
			n.Statements.calc(m, fun, childScope)
			return fun
		})
		return
	} else {
		globalScope.funcDefFuncs = append(globalScope.funcDefFuncs, func() {
			psn := n.Params
			ps := []*ir.Param{}
			for _, v := range psn.Params {
				p := v
				tp, err := p.TP.calc(globalScope)
				if err != nil {
					panic(err)
				}
				param := ir.NewParam(p.ID, tp)
				ps = append(ps, param)
			}
			tp, err := n.RetType.calc(globalScope)
			if err != nil {
				panic(err)
			}
			globalScope.addVar(n.ID, &variable{ir.NewFunc(n.ID, tp, ps...)})
		})
	}
}

func (n *FuncNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
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
	fn := m.NewFunc(n.ID, tp, ps...)
	n.Fn = fn
	b := fn.NewBlock("")
	childScope.block = b

	n.DefaultBlock = b
	for i, v := range ps {
		ptr := b.NewAlloca(v.Type())
		store(v, ptr, childScope)
		childScope.addVar(psn.Params[i].ID, &variable{ptr})
	}

	s.addVar(n.ID, &variable{n.Fn})

	n.Statements.calc(m, fn, childScope)
	return fn
}

type CallFuncNode struct {
	Params   []Node
	FnNode   Node
	Generics []TypeNode
}

func (n *CallFuncNode) calc(m *ir.Module, f *ir.Func, s *scope) value.Value {
	varNode := n.FnNode.(*VarBlockNode)
	fnNode := varNode
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
	var fn *ir.Func

	params := []value.Value{}
	poff := 0
	if fnNode != varNode {
		alloca := deReference(varNode.calc(m, f, s), s)
		name := strings.Trim(alloca.Type().String(), "*%")
		name = name + "." + fnNode.Token
		var err error
		var fnv value.Value
		if len(n.Generics) > 0 {
			if gfn, ok := globalScope.genericFuncs[name]; ok {
				fnv = gfn(m, s, n.Generics...)
			} else {
				panic(fmt.Errorf("cannot find generic method %s", name))
			}
		} else {
			var va *variable
			va, err = s.searchVar(name)
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
			if gfn, ok := globalScope.genericFuncs[fnNode.Token]; ok {
				fn = gfn(m, s, n.Generics...).(*ir.Func)
			} else {
				panic(fmt.Errorf("cannot find generic method %s", fnNode.Token))
			}
		} else {
			fn = fnNode.calc(m, f, s).(*ir.Func)
		}
	}
	for i, v := range n.Params {
		tp := fn.Params[i+poff].Typ
		v2 := v.calc(m, f, s)
		v1 := loadIfVar(v2, s)
		p, err := implicitCast(v1, tp, s)
		if err != nil {
			panic(err)
		}
		params = append(params, p)
	}
	re := s.block.NewCall(fn, params...)
	if re.Type().Equal(types.Void) {
		return re
	}
	// autoAlloc()
	alloc := s.block.NewAlloca(re.Type())
	store(re, alloc, s)
	if fnNode.Token == "heapalloc" {
		mallocTable[alloc] = true
	}
	return alloc
}
