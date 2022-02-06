package ast

import (
	"fmt"

	"github.com/Chronostasys/calculator_go/lexer"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

var getUnsafeCast func(tpin, tpout types.Type, s *scope) value.Value

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
	globalScope.addVar(f.Name(), f)

	gf := m.NewGlobalDef("strf", constant.NewCharArrayFromString("%f\n\x00"))
	p = ir.NewParam("i", types.Double)
	f = m.NewFunc("printFloatln", types.Void, p)
	b = f.NewBlock("")
	// d := b.NewFPExt(p, types.Double)
	b.NewCall(printf, constant.NewGetElementPtr(gf.Typ.ElemType, gf, zero, zero), p)
	b.NewRet(nil)
	globalScope.addVar(f.Name(), f)

	p = ir.NewParam("i", types.I1)
	f = m.NewFunc("printBoolln", types.Void, p)
	b = f.NewBlock("")
	i := b.NewZExt(p, lexer.DefaultIntType())
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), i)
	b.NewRet(nil)
	globalScope.addVar(f.Name(), f)

	p = ir.NewParam("i", lexer.DefaultIntType())
	f = m.NewFunc("unsafemalloc", types.I8Ptr, p)
	globalScope.addVar(f.Name(), f)

	p = ir.NewParam("i", types.I8Ptr)
	f = m.NewFunc("unsafefree", types.Void, p)
	globalScope.addVar(f.Name(), f)

	getUnsafeCast = func(tpin, tpout types.Type, s *scope) value.Value {
		fnname := fmt.Sprintf("unsafecast<%s,%s>", tpin.String(), tpout.String())
		fn, err := globalScope.searchVar(fnname)
		if err != nil {
			p = ir.NewParam("i", tpin)
			f = m.NewFunc(fnname, tpout, p)
			b = f.NewBlock("")
			b.NewBitCast(p, tpout)
			globalScope.addVar(f.Name(), f)
			fn = f
		}
		return fn

	}

}
