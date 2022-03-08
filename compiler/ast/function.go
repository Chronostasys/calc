package ast

import (
	"fmt"
	"strings"

	"github.com/Chronostasys/calc/compiler/helper"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type ParamNode struct {
	ID  string
	TP  TypeNode
	Val value.Value
}

func (n *ParamNode) travel(f func(Node) bool) {
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

func (n *ParamsNode) travel(f func(Node) bool) {
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
	Async      bool
	generator  bool
	i          int
}

func (n *FuncNode) AddtoScope(s *Scope) {
	lableid := 0
	if n.Async {
		n.generator = true
	}
	if n.Statements != nil {
		n.Statements.travel(func(no Node) bool {
			switch node := no.(type) {
			case *YieldNode:
				n.generator = true
				lableid++
				node.label = fmt.Sprintf(".yield%d", lableid)
			case *AwaitNode:
				if !n.Async {
					panic("await only allowed in async func")
				}
				node.label = fmt.Sprintf(".yield%d", lableid)
				n.generator = true
				lableid++
			case *RetNode:
				node.async = n.Async
			case *InlineFuncNode:
				return false
			}
			return true
		})
	}

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
				sig += tp.String() + ","
			}
			gen1 := make(map[string]types.Type)
			for k, v := range s.genericMap {
				gen1[k] = v
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
			gs := s.generics
			defer func() {
				s.generics = gs
			}()

			asyncFunc[s.getFullName(sig)] = n.Async
			s.globalScope.addVar(sig, &variable{v: fun, generics: s.generics})
			b := fun.NewBlock("")
			childScope := s.addChildScope(b)
			childScope.freeFunc = nil
			childScope.genericMap = gen1
			childScope.generics = s.generics
			if n.generator {
				tpname, rtp, idxmap, blockAddrId, context := buildGenaratorCtx(
					n.Statements, n.RetType, s, ps, childScope)
				buildGenerator(rtp, ps, s, childScope, tpname,
					blockAddrId, idxmap, context, tp, n.Statements, n.Async)
			} else {
				for i, v := range ps {
					ptr := gcmalloc(m, childScope, &calcedTypeNode{v.Type()}) // TODO: escape analysis; alloc on heap to avoid captured by inner closure.
					store(v, ptr, childScope)
					childScope.addVar(psn.Params[i].ID, &variable{v: ptr})
				}
				n.Statements.calc(m, fun, childScope)
			}
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
			fullname := n.ID
			if n.Statements != nil {
				fullname = s.getFullName(n.ID)
			}
			asyncFunc[s.getFullName(n.ID)] = n.Async
			s.globalScope.addVar(n.ID, &variable{v: ir.NewFunc(fullname, tp, ps...)})
		})
	}
}

func (n *FuncNode) travel(f func(Node) bool) {
	f(n)
	n.Params.travel(f)
	if n.Statements != nil {
		n.Statements.travel(f)
	}
}

var gencount = 0

func buildGenaratorCtx(st Node, ret TypeNode, s *Scope, ps []*ir.Param, chs *Scope) (
	tpname string,
	rtp types.Type,
	idxmap map[*ir.Param]int, blockAddrId int, context *ctx) {

	idxmap = map[*ir.Param]int{}
	tps, c := buildCtx(st.(*SLNode), chs, []types.Type{}, ps)
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
	sta Node, async bool) {

	// 原理见https://mapping-high-level-constructs-to-llvm-ir.readthedocs.io/en/latest/advanced-constructs/generators.html
	stp := rtp
	gencount++
	b := childScope.block
	t := tp.(*interf)
	if t.interfaceFuncs["GetCurrent"] == nil {
		async = true
	}
	if async { // GetMutex方法
		fname := "GetMutex"

		gcname := s.getFullName(tpname + "." + fname)
		p := ir.NewParam("ctx4", types.NewPointer(rtp))

		s.genericMap = t.genericMaps
		tt, _ := t.interfaceFuncs[fname].RetType.calc(s)
		getcurrent := s.m.NewFunc(gcname, tt, p)
		s.globalScope.addVar(gcname, &variable{v: getcurrent})
		gcentry := getcurrent.NewBlock("")
		chs := s.addChildScope(gcentry)
		retptr := gcentry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
			1,
		)))
		gcentry.NewRet(loadIfVar(retptr, chs))

		fname = "GetContinuous"

		gcname = s.getFullName(tpname + "." + fname)
		p = ir.NewParam("ctx5", types.NewPointer(rtp))

		s.genericMap = t.genericMaps
		tt, _ = t.interfaceFuncs[fname].RetType.calc(s)
		getcurrent = s.m.NewFunc(gcname, tt, p)
		s.globalScope.addVar(gcname, &variable{v: getcurrent})
		gcentry = getcurrent.NewBlock("")
		chs = s.addChildScope(gcentry)
		retptr = gcentry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
			0,
		)))
		i := ScopeMap[CORO_SM_MOD].getStruct("StateMachine").structType
		gcentry.NewRet(gcentry.NewIntToPtr(loadIfVar(retptr, chs), types.NewPointer(i)))

		fname = "IsDone"

		gcname = s.getFullName(tpname + "." + fname)
		p = ir.NewParam("ctx6", types.NewPointer(rtp))

		s.genericMap = t.genericMaps
		tt, _ = t.interfaceFuncs[fname].RetType.calc(s)
		getcurrent = s.m.NewFunc(gcname, tt, p)
		s.globalScope.addVar(gcname, &variable{v: getcurrent})
		gcentry = getcurrent.NewBlock("")
		chs = s.addChildScope(gcentry)
		retptr = gcentry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
			2,
		)))

		gcentry.NewRet(loadIfVar(retptr, chs))

		fname = "SetDone"

		gcname = s.getFullName(tpname + "." + fname)
		p = ir.NewParam("ctx7", types.NewPointer(rtp))

		s.genericMap = t.genericMaps
		tt, _ = t.interfaceFuncs[fname].RetType.calc(s)
		getcurrent = s.m.NewFunc(gcname, tt, p)
		s.globalScope.addVar(gcname, &variable{v: getcurrent})
		gcentry = getcurrent.NewBlock("")
		chs = s.addChildScope(gcentry)
		retptr = gcentry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
			2,
		)))
		store(constant.True, retptr, chs)
		gcentry.NewRet(nil)
	}

	// 生成generator的StepNext函数
	snname := s.getFullName(tpname + "." + "StepNext")
	p := ir.NewParam("ctx1", types.NewPointer(rtp))
	stepNext := s.m.NewFunc(snname, types.I1, p)
	s.globalScope.addVar(snname, &variable{v: stepNext})
	entry := stepNext.NewBlock("")
	generatorScope := s.addChildScope(entry)
	ret := entry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
		blockAddrId+1,
	)))
	nextBlock := entry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
		blockAddrId,
	)))
	sti := entry.NewGetElementPtr(stp, p, zero, zero)

	generatorScope.continueTask = loadIfVar(sti, generatorScope)
	generatorScope.yieldBlock = nextBlock
	generatorScope.yieldRet = ret

	for _, v := range ps { // 取出函数的参数
		ptr := entry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
			idxmap[v],
		)))
		if v.LocalIdent.LocalName == ".closure" { // a closure generator
			// 转移闭包相关数据
			generatorScope.closure = childScope.closure
			generatorScope.trampolineObj = entry.NewBitCast(loadIfVar(ptr, generatorScope), childScope.trampolineObj.Type())
			generatorScope.trampolineVars = childScope.trampolineVars
		}
		generatorScope.addVar(v.LocalName, &variable{v: ptr})
	}

	context.setVals(p, generatorScope)
	realentry := stepNext.NewBlock("entry")

	generatorScope.block = realentry

	// 生成函数体代码
	sta.calc(s.m, stepNext, generatorScope)

	// 闭包清理
	if childScope.freeFunc != nil {
		childScope.freeFunc(generatorScope)
	}
	generatorScope.block.NewRet(constant.False)

	entry.NewIndirectBr(&blockAddress{Value: loadIfVar(nextBlock, &Scope{block: entry})},
		stepNext.Blocks...)

	// 生成generator的GetCurrent/getresult函数
	fname := "GetCurrent"
	if t.interfaceFuncs[fname] == nil {
		fname = "GetResult"
	}
	gcname := s.getFullName(tpname + "." + fname)
	p = ir.NewParam("ctx2", types.NewPointer(rtp))

	s.genericMap = t.genericMaps
	tt, _ := t.interfaceFuncs[fname].RetType.calc(s)
	getcurrent := s.m.NewFunc(gcname, tt, p)
	s.globalScope.addVar(gcname, &variable{v: getcurrent})
	gcentry := getcurrent.NewBlock("")
	chs := s.addChildScope(gcentry)
	retptr := gcentry.NewGetElementPtr(stp, p, zero, constant.NewInt(types.I32, int64(
		blockAddrId+1,
	)))
	gcentry.NewRet(loadIfVar(retptr, chs))

	// generator setup方法
	st := gcmalloc(s.m, childScope, &calcedTypeNode{stp})
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
	if async { // 初始化mutex
		newmu, _ := ScopeMap[CORO_SYNC_MOD].searchVar("NewMutex")
		mu := childScope.block.NewCall(newmu.v)
		retptr := childScope.block.NewGetElementPtr(stp, st, zero, constant.NewInt(types.I32, int64(
			1,
		)))
		store(mu, retptr, childScope)
	}

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
	for _, v := range psn.Params {
		param := v.calc(m, f, s).(*ir.Param)
		ps = append(ps, param)
	}
	tp, err := n.RetType.calc(s)
	if err != nil {
		panic(err)
	}
	// only declaration
	if n.Statements == nil {

		return m.NewFunc(n.ID, tp, ps...)
	}
	fn := m.NewFunc(s.getFullName(n.ID), tp, ps...)
	b := fn.NewBlock("")
	childScope := s.addChildScope(b)
	childScope.freeFunc = nil

	if n.generator {
		tpname, rtp, idxmap, blockAddrId, context := buildGenaratorCtx(
			n.Statements, n.RetType, s, ps, childScope)
		buildGenerator(rtp, ps, s, childScope, tpname,
			blockAddrId, idxmap, context, tp, n.Statements, n.Async)
	} else {
		for i, v := range ps {
			ptr := gcmalloc(m, childScope, &calcedTypeNode{v.Type()}) // TODO: escape analysis; alloc on heap to avoid captured by inner closure.
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

func (n *CallFuncNode) travel(f func(Node) bool) {
	f(n)
	for _, v := range n.Params {
		v.travel(f)
	}
	if n.Next != nil {
		n.Next.travel(f)
	}
	n.FnNode.travel(f)
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
	prev := fnNode.Next
	if !ok {
		scope = s
		prev = fnNode
	}
	fnNode = prev
	if fnNode.Next != nil {
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
	} else {
		fnNode = varNode
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
		oris := name
		idx := strings.Index(name, "<")
		if idx > -1 {
			name = name[:idx]
		}
		oldname := name
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
		i, ok := loadElmType(alloca.Type()).(*interf)
		ok2 := false
		if ok && i.interfaceFuncs != nil {
			_, ok2 = i.interfaceFuncs[fnNode.Token]
		}
		member := false
		if ok2 {
			id := i.interfaceFuncs[fnNode.Token].i
			interger := s.block.NewGetElementPtr(i.Type,
				alloca, zero, constant.NewInt(types.I32, int64(id)))

			v := i.interfaceFuncs[fnNode.Token]
			ret, err := v.RetType.calc(s)
			if err != nil {
				panic(err)
			}
			alloca = deReference(alloca, s)
			in := s.block.NewGetElementPtr(i.Type, alloca, zero, zero)
			alloca = s.block.NewIntToPtr(loadIfVar(in, s), types.I8Ptr)
			ps := []types.Type{types.I8Ptr}
			for _, v := range v.Params.Params {
				p, err := v.TP.calc(s)
				if err != nil {
					panic(err)
				}
				ps = append(ps, p)
			}

			ft := types.NewFunc(ret, ps...)
			fnv = s.block.NewIntToPtr(loadIfVar(interger, s), types.NewPointer(ft))

		} else if len(n.Generics) > 0 {
			if gfn := scope.getGenericFunc(name); gfn != nil {
				gs := []TypeNode{}
				for _, v := range n.Generics {
					t, _ := v.calc(s)
					gs = append(gs, &calcedTypeNode{t})
				}
				fnv = gfn(m, gs...)
			} else {
				panic(fmt.Errorf("cannot find generic method %s", name))
			}
		} else {
			var va *variable
			va, err = scope.searchVar(name)
			if va == nil {
				name = strings.Replace(name, oldname, oris, 1)
				ss := strings.Split(name, ".")
				ssf := strings.Join(ss[:len(ss)-1], ".")
				sse := ss[len(ss)-1]
				st := scope.getStruct(ssf)
				if st == nil {
					panic(fmt.Sprintf("var %s not found", name))
				}
				idx := st.fieldsIdx[sse]
				va = &variable{}
				err = nil
				va.v = s.block.NewGetElementPtr(st.structType, alloca, zero, constant.NewInt(types.I32, int64(idx.idx)))
				member = true
			}
			fnv = va.v
			if err != nil {
				panic(err)
			}
		}
		fn = fnv
		fntp = loadElmType(fn.Type()).(*types.FuncType)
		if len(fntp.Params) != 0 && !member {
			if _, ok := fntp.Params[0].(*types.PointerType); ok {
				alloca = deReference(alloca, s)
			} else {
				alloca = loadIfVar(alloca, s)
			}
			params = append(params, alloca)
			poff = 1
		}
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
				gs := []TypeNode{}
				for _, v := range n.Generics {
					t, _ := v.calc(s)
					gs = append(gs, &calcedTypeNode{t})
				}
				fn = gfn(m, gs...)
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
		var alloc value.Value
		if externMap[fnNode.Token] || stackallocfn[fnNode.Token] {
			alloc = stackAlloc(s.m, s, re.Type())
		} else {
			alloc = gcmalloc(m, s, &calcedTypeNode{re.Type()})
		}
		// alloc := s.block.NewAlloca(re.Type())
		store(re, alloc, s)
		// if fnNode.Token == "heapalloc" {
		// 	mallocTable[alloc] = true
		// }
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
	name := strings.Trim(fn.Ident(), "\"@")
	// tpname := strings.Trim(fn.Type().String(), "%*")
	if asyncFunc[name] || asyncInlineFunc[fn.Type()] {
		i := ScopeMap[CORO_SM_MOD].getStruct("StateMachine").structType

		// statemachie入队列
		vst := loadIfVar(re, s)
		qt, _ := ScopeMap[CORO_MOD].searchVar("QueueTask")
		fqt := qt.v.(*ir.Func)
		c, err := implicitCast(vst, i, s)
		if err != nil {
			panic(err)
		}
		s.block.NewCall(fqt, c)
	}
	s.generics = scope.generics
	return re
}

var asyncFunc = map[string]bool{}
var asyncInlineFunc = map[types.Type]bool{}

var stackallocfn = map[string]bool{
	"sizeof":     true,
	"unsafecast": true,
}

type InlineFuncNode struct {
	Fntype      TypeNode
	Body        Node
	Async       bool
	closureVars map[string]bool
}

func (n *InlineFuncNode) travel(f func(Node) bool) {
	b := f(n)
	if !b {
		return
	}
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
	psm := map[string]bool{}
	for i, v := range fntp.Params {
		id := n.Fntype.(*FuncTypeNode).Args.Params[i].ID
		ps = append(ps, ir.NewParam(id, v))
		psm[id] = true
	}
	fname := fmt.Sprintf("inline.%d", inlinefuncnum)
	cname := fmt.Sprintf("closure%d", inlinefuncnum)

	inlinefuncnum++

	// build closure
	i := 0
	closureArg := ir.NewParam(".closure", types.I8Ptr)
	closureArg.Attrs = append(closureArg.Attrs, enum.ParamAttrNest)
	ps = append([]*ir.Param{closureArg}, ps...)
	fn := m.NewFunc(fname, fntp.RetType, ps...)
	b := fn.NewBlock("")
	chs := s.addChildScope(b)
	chs.freeFunc = nil
	chs.closure = true
	fields := []types.Type{}
	vals := []value.Value{}
	for k := range n.closureVars {
		if psm[k] {
			// skip params
			delete(n.closureVars, k)
		}
		fullname := s.getFullName(k)
		if _, ok := s.globalScope.genericFuncs[fullname]; ok {
			// skip generic func
			delete(n.closureVars, k)
		}
		if _, ok := s.globalScope.vartable[fullname]; ok {
			// skip global funcs
			delete(n.closureVars, k)
		}
		if _, ok := ScopeMap[k]; ok {
			// skip module
			delete(n.closureVars, k)
		}
		if _, ok := externMap[k]; ok {
			// skip extern func
			delete(n.closureVars, k)
		}
	}
	// HACK: 内部匿名函数的入参可能在外边找不到
	dels := []string{}
	for k := range n.closureVars {
		va, _ := s.searchVar(k)
		if va == nil {
			dels = append(dels, k)
		}
	}
	for _, v := range dels {
		delete(n.closureVars, v)
	}

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
	trampInit, _ := s.globalScope.searchVar("llvm.init.trampoline")
	trampAdj, _ := s.globalScope.searchVar("llvm.adjust.trampoline")
	enableExe, _ := s.globalScope.searchVar("__enable_execute_stack")
	// 72 bytes and 16 align, see https://stackoverflow.com/questions/15509341/how-much-space-for-a-llvm-trampoline
	tramptp := types.NewArray(72, types.I8)
	tramp := gcmalloc(m, s, &calcedTypeNode{tramptp}) // alloc on heap to avoid call it in another thread
	tramp1 := s.block.NewGetElementPtr(tramptp, tramp, zero, zero)

	allo := gcmalloc(m, s, &calcedTypeNode{st})
	for i, v := range vals {
		ptr := s.block.NewGetElementPtr(st, allo, zero,
			constant.NewInt(types.I32, int64(i)))
		store(v, ptr, s)
	}
	cloCast := s.block.NewBitCast(allo, types.I8Ptr)
	fncast := s.block.NewBitCast(fn, types.I8Ptr)
	s.block.NewCall(trampInit.v, tramp1, fncast, cloCast)
	fnptr := s.block.NewCall(trampAdj.v, tramp1)
	s.block.NewCall(enableExe.v, fnptr)
	var tp types.Type = types.NewPointer(fntp)
	if n.Async { // 记录async方法
		asyncInlineFunc[tp] = n.Async
	}
	fun := s.block.NewBitCast(fnptr, tp)
	chs.trampolineObj = chs.block.NewBitCast(closureArg, allo.Type())
	// chs.freeFunc = func(s *Scope) { // make closure var gcable
	// 	store(constant.NewNull(allo.Type().(*types.PointerType)), g, s)
	// }

	lableid := 0
	generator := false
	n.Body.travel(func(no Node) bool {
		switch node := no.(type) {
		case *YieldNode:
			generator = true
			lableid++
			node.label = fmt.Sprintf(".yield%d", lableid)
		case *AwaitNode:
			if !n.Async {
				panic("await only allowed in async func")
			}
			node.label = fmt.Sprintf(".yield%d", lableid)
			generator = true
			lableid++
		case *RetNode:
			node.async = n.Async
		case defNode:
			node.setVal(nil)
		case *InlineFuncNode:
			return false
		}
		return true

	})

	if generator {
		tpname, rtp, idxmap, blockAddrId, context := buildGenaratorCtx(
			n.Body, n.Fntype.(*FuncTypeNode).Ret, s, ps, chs)
		buildGenerator(rtp, ps, s, chs, tpname,
			blockAddrId, idxmap, context, fn.Sig.RetType, n.Body, n.Async)
	} else {
		for i, v := range ps {
			ptr := gcmalloc(m, chs, &calcedTypeNode{v.Type()}) // TODO: escape analysis; alloc on heap to avoid captured by inner closure.
			store(v, ptr, chs)
			chs.addVar(ps[i].LocalName, &variable{v: ptr})
		}
		n.Body.calc(m, fn, chs)
	}

	return fun
}
