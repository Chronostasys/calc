package lexer

import (
	"math/bits"

	"github.com/llir/llvm/ir/types"
)

func DefaultIntType() *types.IntType {
	return types.NewInt(bits.UintSize)
}

func DefaultFloatType() *types.FloatType {
	if bits.UintSize == 64 {
		return types.Double
	}
	return types.Float
}
