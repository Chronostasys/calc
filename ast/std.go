package ast

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

func AddSTDFunc(m *ir.Module) {
	printf := m.NewFunc("printf", types.I32, ir.NewParam("formatstr", types.I8Ptr))
	printf.Sig.Variadic = true
	gi := m.NewGlobalDef("stri", constant.NewCharArrayFromString("%d\n\x00"))
	p := ir.NewParam("i", types.I32)
	f := m.NewFunc("printIntln", types.Void, p)
	b := f.NewBlock("")
	zero := constant.NewInt(types.I32, 0)
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), p)
	b.NewRet(nil)
	fntable[f.Name()] = &FuncNode{Fn: f, ID: f.Name()}

	gf := m.NewGlobalDef("strf", constant.NewCharArrayFromString("%f\n\x00"))
	p = ir.NewParam("i", types.Float)
	f = m.NewFunc("printFloatln", types.Void, p)
	b = f.NewBlock("")
	d := b.NewFPExt(p, types.Double)
	b.NewCall(printf, constant.NewGetElementPtr(gf.Typ.ElemType, gf, zero, zero), d)
	b.NewRet(nil)
	fntable[f.Name()] = &FuncNode{Fn: f, ID: f.Name()}

	p = ir.NewParam("i", types.I1)
	f = m.NewFunc("printBoolln", types.Void, p)
	b = f.NewBlock("")
	i32 := b.NewZExt(p, types.I32)
	b.NewCall(printf, constant.NewGetElementPtr(gi.Typ.ElemType, gi, zero, zero), i32)
	b.NewRet(nil)
	fntable[f.Name()] = &FuncNode{Fn: f, ID: f.Name()}
}
