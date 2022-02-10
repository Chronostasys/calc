package ast

import (
	"fmt"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

func AddSTDFunc(m *ir.Module, s *Scope) {
	printf := m.NewFunc("printf", types.I32, ir.NewParam("formatstr", types.I8Ptr))
	printf.Sig.Variadic = true
	s.globalScope.addVar(printf.Name(), &variable{printf})
	gi := m.NewGlobalDef("stri", constant.NewCharArrayFromString("%d\n\x00"))
	p := ir.NewParam("i", lexer.DefaultIntType())
	f := m.NewFunc("printIntln", types.Void, p)
	b := f.NewBlock("")
	zero := constant.NewInt(types.I32, 0)
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), p)
	b.NewRet(nil)
	s.globalScope.addVar(f.Name(), &variable{f})

	gf := m.NewGlobalDef("strf", constant.NewCharArrayFromString("%f\n\x00"))
	p = ir.NewParam("i", types.Double)
	f = m.NewFunc("printFloatln", types.Void, p)
	b = f.NewBlock("")
	// d := b.NewFPExt(p, types.Double)
	b.NewCall(printf, constant.NewGetElementPtr(gf.Typ.ElemType, gf, zero, zero), p)
	b.NewRet(nil)
	s.globalScope.addVar(f.Name(), &variable{f})

	p = ir.NewParam("i", types.I1)
	f = m.NewFunc("printBoolln", types.Void, p)
	b = f.NewBlock("")
	i := b.NewZExt(p, lexer.DefaultIntType())
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), i)
	b.NewRet(nil)
	s.globalScope.addVar(f.Name(), &variable{f})

	p = ir.NewParam("i", lexer.DefaultIntType())
	f = m.NewFunc("GC_malloc", types.I8Ptr, p)
	s.globalScope.addVar(f.Name(), &variable{f})

	p = ir.NewParam("i", lexer.DefaultIntType())
	f = m.NewFunc("Sleep", lexer.DefaultIntType(), p)
	s.globalScope.addVar(f.Name(), &variable{f})

	p = ir.NewParam("i", types.I8Ptr)
	f = m.NewFunc("free", types.Void, p)
	s.globalScope.addVar(f.Name(), &variable{f})

	p = ir.NewParam("i", types.I8Ptr)
	f = m.NewFunc("memset", types.I8Ptr, p, ir.NewParam("v", lexer.DefaultIntType()), ir.NewParam("len", lexer.DefaultIntType()))
	s.globalScope.addVar(f.Name(), &variable{f})

	p1 := ir.NewParam("dst", types.I8Ptr)
	p2 := ir.NewParam("src", types.I8Ptr)
	f = m.NewFunc("memcpy", types.I8Ptr, p1, p2, ir.NewParam("len", lexer.DefaultIntType()))
	s.globalScope.addVar(f.Name(), &variable{f})

	s.globalScope.addGeneric("unsafecast", func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value {
		tpin, _ := gens[0].calc(s)
		tpout, _ := gens[1].calc(s)
		fnname := fmt.Sprintf("unsafecast<%s,%s>", tpin.String(), tpout.String())
		fn, err := s.globalScope.searchVar(fnname)
		if err != nil {
			p = ir.NewParam("i", tpin)
			f = m.NewFunc(fnname, tpout, p)
			b = f.NewBlock("")
			cast := b.NewBitCast(p, tpout)
			b.NewRet(cast)
			fn = &variable{f}
			s.globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

	// sizeof see https://stackoverflow.com/questions/14608250/how-can-i-find-the-size-of-a-type
	s.globalScope.addGeneric("sizeof", func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value {
		tp, _ := gens[0].calc(s)
		fnname := fmt.Sprintf("sizeof<%s>", tp.String())
		fn, err := s.globalScope.searchVar(fnname)
		if err != nil {
			f = m.NewFunc(fnname, lexer.DefaultIntType())
			b = f.NewBlock("")
			sizePtr := b.NewGetElementPtr(tp, constant.NewNull(types.NewPointer(tp)), constant.NewInt(lexer.DefaultIntType(), 1))
			size := b.NewPtrToInt(sizePtr, lexer.DefaultIntType())
			b.NewRet(size)
			fn = &variable{f}
			s.globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

	s.globalScope.addGeneric("ptrtoint", func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value {
		tp, _ := gens[0].calc(s)
		fnname := fmt.Sprintf("ptrtoint<%s>", tp.String())
		fn, err := s.globalScope.searchVar(fnname)
		if err != nil {
			p := ir.NewParam("ptr", tp)
			f = m.NewFunc(fnname, lexer.DefaultIntType(), p)
			b = f.NewBlock("")
			ptr := b.NewPtrToInt(p, lexer.DefaultIntType())
			b.NewRet(ptr)
			fn = &variable{f}
			s.globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

	s.globalScope.addGeneric("inttoptr", func(m *ir.Module, s *Scope, gens ...TypeNode) value.Value {
		tp, _ := gens[0].calc(s)
		fnname := fmt.Sprintf("inttoptr<%s>", tp.String())
		fn, err := s.globalScope.searchVar(fnname)
		if err != nil {
			p := ir.NewParam("int", lexer.DefaultIntType())
			f = m.NewFunc(fnname, tp, p)
			b = f.NewBlock("")
			ptr := b.NewIntToPtr(p, tp)
			b.NewRet(ptr)
			fn = &variable{f}
			s.globalScope.addVar(f.Name(), fn)
		}
		return fn.v

	})

}
