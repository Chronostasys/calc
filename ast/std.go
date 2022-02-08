package ast

import (
	"fmt"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func AddSTDFunc(m *ir.Module) {
	printf := m.NewFunc("printf", types.I32, ir.NewParam("formatstr", types.I8Ptr))
	printf.Sig.Variadic = true
	gi := m.NewGlobalDef("stri", constant.NewCharArrayFromString("%d\n\x00"))
	p := ir.NewParam("i", lexer.DefaultIntType())
	f := m.NewFunc("printIntln", types.Void, p)
	b := f.NewBlock("")
	zero := constant.NewInt(types.I32, 0)
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), p)
	b.NewRet(nil)
	globalScope.addVar(f.Name(), &variable{f, &varheap{}})

	gf := m.NewGlobalDef("strf", constant.NewCharArrayFromString("%f\n\x00"))
	p = ir.NewParam("i", types.Double)
	f = m.NewFunc("printFloatln", types.Void, p)
	b = f.NewBlock("")
	// d := b.NewFPExt(p, types.Double)
	b.NewCall(printf, constant.NewGetElementPtr(gf.Typ.ElemType, gf, zero, zero), p)
	b.NewRet(nil)
	globalScope.addVar(f.Name(), &variable{f, &varheap{}})

	p = ir.NewParam("i", types.I1)
	f = m.NewFunc("printBoolln", types.Void, p)
	b = f.NewBlock("")
	i := b.NewZExt(p, lexer.DefaultIntType())
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), i)
	b.NewRet(nil)
	globalScope.addVar(f.Name(), &variable{f, &varheap{}})

	p = ir.NewParam("i", lexer.DefaultIntType())
	f = m.NewFunc("malloc", types.I8Ptr, p)
	globalScope.addVar(f.Name(), &variable{f, &varheap{}})

	p = ir.NewParam("i", types.I8Ptr)
	f = m.NewFunc("free", types.Void, p)
	globalScope.addVar(f.Name(), &variable{f, &varheap{}})

	p = ir.NewParam("i", types.I8Ptr)
	f = m.NewFunc("memset", types.I8Ptr, p, ir.NewParam("v", lexer.DefaultIntType()), ir.NewParam("len", lexer.DefaultIntType()))
	globalScope.addVar(f.Name(), &variable{f, &varheap{}})

	globalScope.addGeneric("unsafecast", func(m *ir.Module, s *scope, gens ...TypeNode) value.Value {
		tpin, _ := gens[0].calc(s)
		tpout, _ := gens[1].calc(s)
		fnname := fmt.Sprintf("unsafecast<%s,%s>", tpin.String(), tpout.String())
		fn, err := globalScope.searchVar(fnname)
		if err != nil {
			p = ir.NewParam("i", tpin)
			f = m.NewFunc(fnname, tpout, p)
			b = f.NewBlock("")
			cast := b.NewBitCast(p, tpout)
			b.NewRet(cast)
			fn = &variable{f, &varheap{}}
			globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

	// sizeof see https://stackoverflow.com/questions/14608250/how-can-i-find-the-size-of-a-type
	globalScope.addGeneric("sizeof", func(m *ir.Module, s *scope, gens ...TypeNode) value.Value {
		tp, _ := gens[0].calc(s)
		fnname := fmt.Sprintf("sizeof<%s>", tp.String())
		fn, err := globalScope.searchVar(fnname)
		if err != nil {
			f = m.NewFunc(fnname, lexer.DefaultIntType())
			b = f.NewBlock("")
			sizePtr := b.NewGetElementPtr(tp, constant.NewNull(types.NewPointer(tp)), constant.NewInt(lexer.DefaultIntType(), 1))
			size := b.NewPtrToInt(sizePtr, lexer.DefaultIntType())
			b.NewRet(size)
			fn = &variable{f, &varheap{}}
			globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

	globalScope.addGeneric("sizeofwraped", func(m *ir.Module, s *scope, gens ...TypeNode) value.Value {
		tp, _ := gens[0].calc(s)
		fnname := fmt.Sprintf("sizeofwraped<%s>", tp.String())
		fn, err := globalScope.searchVar(fnname)
		if err != nil {
			f = m.NewFunc(fnname, lexer.DefaultIntType())
			b = f.NewBlock("")
			tp = types.NewStruct(types.I8, tp)
			sizePtr := b.NewGetElementPtr(tp, constant.NewNull(types.NewPointer(tp)), constant.NewInt(lexer.DefaultIntType(), 1))
			size := b.NewPtrToInt(sizePtr, lexer.DefaultIntType())
			b.NewRet(size)
			fn = &variable{f, &varheap{}}
			globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

}
