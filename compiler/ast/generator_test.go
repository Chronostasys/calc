package ast

import (
	"fmt"
	"testing"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
)

func Test_buildCtx(t *testing.T) {
	type args struct {
		sl  *SLNode
		s   *Scope
		tps []types.Type
	}
	arg := args{
		s:   NewGlobalScope(ir.NewModule()),
		tps: []types.Type{},
		sl: &SLNode{
			Children: []Node{
				&DefAndAssignNode{ID: "x", ValNode: &NumNode{zero}},
				&IfNode{BoolExp: &BoolConstNode{Val: true}, Statements: &SLNode{
					Children: []Node{
						&DefAndAssignNode{ID: "x", ValNode: &NumNode{zero}},
					},
				}},
			},
		},
	}
	tps, c := buildCtx(arg.sl, arg.s, arg.tps, nil)
	if len(tps) != 2 {
		t.Error("expect 2 fields, got", len(tps))
	}
	if _, ok := tps[1].(*types.StructType); !ok {
		t.Error("2nd field should be structtype, got", tps[1])
	}
	fmt.Println(c)
}
