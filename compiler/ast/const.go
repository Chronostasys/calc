package ast

const (
	INDEX_RELOAD     = "IndexOp"
	INDEX_SET_RELOAD = "IndexSetOp"
	CORO_MOD         = "github.com/Chronostasys/calc/runtime/coro"
	GEN_MOD          = "github.com/Chronostasys/calc/runtime/generator"
	CORO_SM_MOD      = CORO_MOD + "/sm"
	CORO_SYNC_MOD    = "github.com/Chronostasys/calc/runtime/coro/sync"
	LIBUV            = "github.com/Chronostasys/calc/runtime/libuv"
	SLICE            = "github.com/Chronostasys/calc/runtime/slice"
	RUNTIME          = "github.com/Chronostasys/calc/runtime"
)

var DIAGNOSTIC_SOURCE = "Calc lsp"
