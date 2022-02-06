package ast

import "github.com/llir/llvm/ir/types"

type interf struct {
	*types.IntType
	interfaceFuncs map[string]*FuncNode
	innerType      types.Type
	name           string
}
