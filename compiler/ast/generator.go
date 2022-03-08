package ast

import (
	"github.com/Chronostasys/calc/compiler/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type ctx struct {
	idxmap []*ctx
	id     int
	i      int
	node   defNode
	father *ctx
}

func (c *ctx) setVals(st value.Value, s *Scope) {
	b := s.block
	for _, v := range c.idxmap {
		id := v.id
		if v.node != nil {
			v.node.setVal(func(*Scope) value.Value {
				return b.NewGetElementPtr(loadElmType(st.Type()), st,
					zero, constant.NewInt(types.I32, int64(id)))
			})
		}
		if v.idxmap != nil {
			subst := b.NewGetElementPtr(loadElmType(st.Type()), st,
				zero, constant.NewInt(types.I32, int64(id)))
			v.setVals(subst, s)
		}
	}
}

func buildCtx(sl *SLNode, s *Scope, tps []types.Type, ps []*ir.Param) ([]types.Type, *ctx) {
	mvart := map[string]map[string]*variable{}
	for k, v := range ScopeMap {
		mvart[k] = map[string]*variable{}
		for k2, v2 := range v.vartable {
			mvart[k][k2] = v2
		}
	}
	defer func() {
		for k, v := range ScopeMap {
			v.vartable = mvart[k]
		}
		s.childrenScopes = nil
	}()
	c := &ctx{idxmap: []*ctx{}}
	var trf func(n Node)
	tpm := ir.NewModule()
	tpf := tpm.NewFunc("xxxx", types.Void)
	tpsc := newScope(tpf.NewBlock(""))
	tpsc.Pkgname = s.Pkgname
	for _, v := range ps {
		tpsc.addVar(v.LocalName, &variable{v: v})
	}
	tpsc.globalScope = s.globalScope
	tpsc.parent = s.parent
	tpsc.m = tpm
	tpsc.trampolineObj = s.trampolineObj
	tpsc.trampolineVars = s.trampolineVars

	for k, v := range s.globalScope.vartable {
		tpsc.vartable[k] = v
	}
	trf = func(n Node) {
		switch node := n.(type) {
		case *IfElseNode:
			ntps, ct := buildCtx(node.Statements.(*SLNode), s.addChildScope(tpf.NewBlock("")), []types.Type{}, ps)
			tps = append(tps, types.NewStruct(ntps...))
			ct.father = c
			ct.id = c.i
			c.idxmap = append(c.idxmap, ct)

			c.i++
		case *IfNode:
			ntps, ct := buildCtx(node.Statements.(*SLNode), s.addChildScope(tpf.NewBlock("")), []types.Type{}, ps)
			tps = append(tps, types.NewStruct(ntps...))
			ct.father = c
			ct.id = c.i
			c.idxmap = append(c.idxmap, ct)
			c.i++
		case *ForNode:
			ntps, ct := buildCtx(node.Statements.(*SLNode), s.addChildScope(tpf.NewBlock("")), []types.Type{}, ps)
			tps = append(tps, types.NewStruct(ntps...))
			ct.father = c
			ct.id = c.i
			c.idxmap = append(c.idxmap, ct)
			c.i++
		case *InlineFuncNode:
			ntps, ct := buildCtx(node.Body.(*SLNode), s, []types.Type{}, ps)
			tps = append(tps, types.NewStruct(ntps...))
			ct.father = c
			ct.i = c.i
			c.idxmap = append(c.idxmap, ct)
			c.i++
		case *DefineNode:
			tp, _ := node.TP.calc(s)
			tpsc.addVar(node.ID, &variable{v: stackAlloc(tpm, tpsc, tp)})
			tps = append(tps, tp)
			c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c, node: node})

			c.i++
		case *DefAndAssignNode:
			var tp types.Type
			tp = node.calc(tpm, tpf, tpsc).Type()
			tpsc.addVar(node.ID, &variable{v: stackAlloc(tpm, tpsc, tp)})
			tps = append(tps, getElmType(tp))
			c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c, node: node})
			c.i++
			if an, ok := node.ValNode.(*AwaitNode); ok { // async statemachine
				tps = append(tps, an.generator)
				c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c, node: an})
				c.i++

			}
		case *AwaitNode: // async statemachine
			node.calc(tpm, tpf, tpsc)
			tps = append(tps, node.generator)
			c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c, node: node})
			c.i++
		default:

		}
	}
	tps = append(tps, lexer.DefaultIntType())
	c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c})
	c.i++
	tps = append(tps, types.NewPointer(ScopeMap[CORO_SYNC_MOD].getStruct("Mutex").structType))
	c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c})
	c.i++
	tps = append(tps, types.I1)
	c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c})
	c.i++
	for _, v := range sl.Children {
		trf(v)
	}
	return tps, c
}

type YieldNode struct {
	Exp   Node
	label string
}

func (n *YieldNode) travel(f func(Node) bool) {
	f(n)
	if n.Exp != nil {
		n.Exp.travel(f)
	}
}

func (n *YieldNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if n.Exp != nil {
		v := n.Exp.calc(m, f, s)
		v, err := implicitCast(loadIfVar(v, s), getElmType(s.yieldRet.Type()), s)
		if err != nil {
			panic(err)
		}
		store(v, s.yieldRet, s)
	}
	nb := f.NewBlock(n.label)
	store(constant.NewBlockAddress(f, nb), s.yieldBlock, s)
	s.block.NewRet(constant.True)
	s.block = nb
	return zero
}

type blockAddress struct {
	value.Value
}

func (c *blockAddress) IsConstant() {
}
