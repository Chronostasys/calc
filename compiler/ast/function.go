package ast

import (
	"fmt"
	"strings"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type ParamNode struct {
	ID  string
	TP  TypeNode
	Val value.Value
}

func (n *ParamNode) travel(f func(Node)) {
	f(n)
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

func (n *ParamsNode) travel(f func(Node)) {
	f(n)
	for _, v := range n.Params {
		v.travel(f)
	}
}

func (n *ParamsNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {

	return zero
}

type FuncNode struct {
	Params     *ParamsNode
	ID         string
	RetType    TypeNode
	Statements Node
	Generics   []string
	generator  bool
}

func (n *FuncNode) AddtoScope(s *Scope) {
	lableid := 0
	n.travel(func(no Node) {
		if node, ok := no.(*YieldNode); ok {
			n.generator = true
			lableid++
			node.label = fmt.Sprintf(".yield%d", lableid)
		}
	})
	if len(n.Generics) > 0 {
		s.globalScope.addGeneric(n.ID, func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value {
			psn := n.Params
			ps := []*ir.Param{}
			sig := fmt.Sprintf("%s<", n.ID)
			defparams := func() {
				s.currParam = 0
				for _, v := range psn.Params {
					p := v
					tp, err := p.TP.calc(s)
					s.currParam++
					if err != nil {
						panic(err)
					}
					param := ir.NewParam(p.ID, tp)
					ps = append(ps, param)
				}
				s.paramGenerics = nil
			}
			if len(gens) == 0 {
				defparams()
			} else {
				old := s.genericMap
				s.genericMap = make(map[string]types.Type)
				defer func() {
					s.genericMap = old
				}()
			}
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
			if len(gens) != 0 {
				defparams()
			}
			fn, err := s.globalScope.searchVar(sig)
			if err == nil {
				return fn.v
			}
			tp, err := n.RetType.calc(s)
			if err != nil {
				panic(err)
			}
			fun := m.NewFunc(s.getFullName(sig), tp, ps...)

			s.globalScope.addVar(sig, &variable{v: fun})
			b := fun.NewBlock("")
			childScope := s.addChildScope(b)

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

func (n *FuncNode) travel(f func(Node)) {
	f(n)
	n.Params.travel(f)
	n.Statements.travel(f)
}

var gencount = 0

func buildGenaratorCtx(st Node, ret TypeNode, s *Scope, ps []*ir.Param) (
	tpname string,
	rtp types.Type,
	idxmap map[*ir.Param]int, blockAddrId int, context *ctx) {

	idxmap = map[*ir.Param]int{}
	tps, c := buildCtx(st.(*SLNode), s, []types.Type{})
	context = c
	for _, v := range ps {
		tps = append(tps, v.Type())
		c.idxmap = append(c.idxmap, &ctx{id: c.i, father: c})
		idxmap[v] = c.i
		c.i++
	}
	blockAddrId = c.i
	tps = append(tps, types.I8Ptr) // next block address
	inner, _ := ret.(*BasicTypeNode).Generics[0].calc(s)
	tps = append(tps, inner) // return value
	rtp = types.NewStruct(tps...)
	tpname = fmt.Sprintf("_%dgeneratorctx", gencount)
	rtp = s.m.NewTypeDef(s.getFullName(tpname), rtp)
	return
}

func buildGenerator(rtp types.Type, ps []*ir.Param,
	s, childScope *Scope, tpname string, blockAddrId int,
	idxmap map[*ir.Param]int, context *ctx, tp types.Type,
	sta Node) {

	// 原理见https://mapping-high-level-constructs-to-llvm-ir.readthedocs.io/en/latest/advanced-constructs/generators.html
	stp := rtp
	gencount++
	b := childScope.block

	// 生成generator的StepNext函数
	snname := s.getFullName(tpname + "." + "StepNext")
	p := ir.NewParam("ctx", types.NewPointer(rtp))
	stepNext := s.m.NewFunc(snname, types.I1, p)
	entry := stepNext.NewBlock("")
	generatorScope := s.addChildScope(entry)
	ret := entry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
		blockAddrId+1,
	)))
	nextBlock := entry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
		blockAddrId,
	)))
	generatorScope.yieldBlock = nextBlock
	generatorScope.yieldRet = ret
	for _, v := range ps { // 取出函数的参数
		ptr := b.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
			idxmap[v],
		)))
		generatorScope.addVar(v.LocalName, &variable{v: ptr})
	}

	context.setVals(p, generatorScope)
	realentry := stepNext.NewBlock("entry")

	generatorScope.block = realentry

	// 转移闭包相关数据
	generatorScope.closure = childScope.closure
	generatorScope.trampolineObjG = childScope.trampolineObjG
	generatorScope.trampolineVars = childScope.trampolineVars

	// 生成函数体代码
	sta.calc(s.m, stepNext, generatorScope)

	// 闭包清理
	if childScope.freeFunc != nil {
		childScope.freeFunc(generatorScope)
	}
	generatorScope.block.NewRet(constant.False)

	entry.NewIndirectBr(&blockAddress{Value: loadIfVar(nextBlock, &Scope{block: entry})},
		stepNext.Blocks...)
	s.globalScope.addVar(snname, &variable{v: stepNext})

	// 生成generator的GetCurrent函数
	gcname := s.getFullName(tpname + "." + "GetCurrent")
	p = ir.NewParam("ctx", types.NewPointer(rtp))
	t := tp.(*interf)
	s.genericMap = t.genericMaps
	tt, _ := t.interfaceFuncs["GetCurrent"].RetType.calc(s)
	getcurrent := s.m.NewFunc(gcname, tt, p)
	gcentry := getcurrent.NewBlock("")
	chs := s.addChildScope(gcentry)
	retptr := gcentry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
		blockAddrId+1,
	)))
	gcentry.NewRet(loadIfVar(retptr, chs))
	s.globalScope.addVar(gcname, &variable{v: getcurrent})

	// generator setup方法
	st := heapAlloc(s.m, childScope, &calcedTypeNode{stp})
	for _, v := range ps { // 保存函数的参数
		ptr := b.NewGetElementPtr(stp, st, zero, constant.NewInt(types.I32, int64(
			idxmap[v],
		)))
		store(v, ptr, childScope)
		// childScope.addVar(psn.Params[i].ID, &variable{v: ptr})
	}
	// 存下一个block地址
	ptr := b.NewGetElementPtr(stp, st, zero, constant.NewInt(types.I32, int64(
		blockAddrId,
	)))
	store(constant.NewBlockAddress(stepNext, realentry), ptr, childScope)
	r, err := implicitCast(st, tp, childScope)
	if err != nil {
		panic(err)
	}
	childScope.block.NewRet(r) // 返回context（即generator）
}

func (n *FuncNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	if len(n.Generics) > 0 {
		// generic function will be generated while call
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

	b := fn.NewBlock("")
	childScope.block = b

	if n.generator {
		tpname, rtp, idxmap, blockAddrId, context := buildGenaratorCtx(
			n.Statements, n.RetType, s, ps)
		buildGenerator(rtp, ps, s, childScope, tpname,
			blockAddrId, idxmap, context, tp, n.Statements)
	} else {
		for i, v := range ps {
			ptr := b.NewAlloca(v.Type())
			store(v, ptr, childScope)
			childScope.addVar(psn.Params[i].ID, &variable{v: ptr})
		}
		n.Statements.calc(m, fn, childScope)
	}
	s.addVar(n.ID, &variable{v: fn})

	if n.ID == "main" {
		s.globalScope.vartable["main"].v = fn
	}
	return fn
}

type CallFuncNode struct {
	Params   []Node
	FnNode   Node
	parent   value.Value
	Next     Node
	Generics []TypeNode
}

func (n *CallFuncNode) tp() TypeNode {
	panic("not impl")
}

func (n *CallFuncNode) travel(f func(Node)) {
	f(n)
	for _, v := range n.Params {
		v.travel(f)
	}
	if n.Next != nil {
		n.Next.travel(f)
	}
}

func (n *CallFuncNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	var fn value.Value
	var fntp *types.FuncType

	params := []value.Value{}
	pvs := []value.Value{}
	poff := 0
	varNode := n.FnNode.(*VarBlockNode)
	fnNode := varNode
	scope, ok := ScopeMap[varNode.Token]
	paramGenerics := [][]types.Type{}
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
	paramGenerics = append(paramGenerics, s.generics)
	for _, v := range n.Params {
		v2 := v.calc(m, f, s)
		v1 := loadIfVar(v2, s)
		pvs = append(pvs, v1)
		paramGenerics = append(paramGenerics, s.generics)
	}
	if n.parent != nil {
		varNode = nil
	}
	if fnNode != varNode {
		var alloca value.Value
		if varNode != nil {
			alloca = deReference(varNode.calc(m, f, s), s)
			paramGenerics[0] = s.generics
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
		scope.paramGenerics = paramGenerics
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
		fn = fnv
		fntp = loadElmType(fn.Type()).(*types.FuncType)
		if _, ok := fntp.Params[0].(*types.PointerType); ok {
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
			scope.paramGenerics = paramGenerics
			if gfn := scope.getGenericFunc(token); gfn != nil {
				fn = gfn(m, n.Generics...)
				fntp = loadElmType(fn.Type()).(*types.FuncType)
			} else {
				panic(fmt.Errorf("cannot find generic method %s", fnNode.Token))
			}
		} else {
			v1 := fnNode.calc(m, f, s)
			fn = loadIfVar(v1, s)
			fntp = loadElmType(fn.Type()).(*types.FuncType)
		}
	}
	for i, v := range pvs {
		tp := fntp.Params[i+poff]
		v1 := v
		p, err := implicitCast(v1, tp, s)
		if err != nil {
			panic(err)
		}
		params = append(params, p)
	}

	var re value.Value = s.block.NewCall(loadIfVar(fn, s), params...)
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

type InlineFuncNode struct {
	Fntype      TypeNode
	Body        Node
	closureVars map[string]bool
}

func (n *InlineFuncNode) travel(f func(Node)) {
	f(n)
	n.Body.travel(f)
}
func (n *InlineFuncNode) tp() TypeNode {
	return n.Fntype
}

var inlinefuncnum int

func (n *InlineFuncNode) calc(m *ir.Module, f *ir.Func, s *Scope) value.Value {
	fnt, err := n.Fntype.calc(s)
	if err != nil {
		panic(err)
	}
	fntp := fnt.(*types.PointerType).ElemType.(*types.FuncType)
	ps := []*ir.Param{}
	for i, v := range fntp.Params {
		ps = append(ps, ir.NewParam(fmt.Sprintf("%d", i), v))
	}
	fname := fmt.Sprintf("inline.%d", inlinefuncnum)
	cname := fmt.Sprintf("closure%d", inlinefuncnum)
	cvarname := fmt.Sprintf("closurevar%d", inlinefuncnum)

	inlinefuncnum++

	// build closure
	i := 0
	fn := m.NewFunc(fname, fntp.RetType, ps...)
	b := fn.NewBlock("")
	chs := s.addChildScope(b)
	chs.closure = true
	fields := []types.Type{}
	vals := []value.Value{}
	for k := range n.closureVars {
		v := &fieldval{}
		v.idx = i
		va, _ := s.searchVar(k)
		v.v = va.v
		chs.trampolineVars[s.getFullName(k)] = v
		i++
		vals = append(vals, v.v)
		fields = append(fields, v.v.Type())
	}
	var st types.Type
	st = types.NewStruct(fields...)
	st = m.NewTypeDef(cname, st)

	// alloc closure captured var
	allo := heapAlloc(m, s, &calcedTypeNode{st})
	for i, v := range vals {
		ptr := s.block.NewGetElementPtr(st, allo, zero,
			constant.NewInt(types.I32, int64(i)))
		store(v, ptr, s)
	}
	g := m.NewGlobalDef(cvarname, constant.NewZeroInitializer(allo.Type()))
	store(allo, g, s)
	chs.trampolineObjG = g
	chs.freeFunc = func(s *Scope) { // make closure var gcable
		store(constant.NewNull(allo.Type().(*types.PointerType)), g, s)
	}

	lableid := 0
	generator := false
	n.travel(func(no Node) {
		if node, ok := no.(*YieldNode); ok {
			generator = true
			lableid++
			node.label = fmt.Sprintf(".yield%d", lableid)
		}
	})

	if generator {
		tpname, rtp, idxmap, blockAddrId, context := buildGenaratorCtx(
			n.Body, n.Fntype.(*FuncTypeNode).Ret, s, ps)
		buildGenerator(rtp, ps, s, chs, tpname,
			blockAddrId, idxmap, context, fn.Sig.RetType, n.Body)
	} else {
		for i, v := range ps {
			ptr := b.NewAlloca(v.Type())
			store(v, ptr, chs)
			chs.addVar(ps[i].LocalName, &variable{v: ptr})
		}
		n.Body.calc(m, fn, chs)
	}

	return fn
}
