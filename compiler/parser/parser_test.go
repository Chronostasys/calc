package parser

import (
	"testing"

	"github.com/llir/llvm/ir"
)

func TestParser_defineAndAssign(t *testing.T) {
	p := NewParser(ir.NewModule())
	p.lexer.SetInput("a := struct{i int}{i:10}")
	_, err := p.defineAndAssign()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
