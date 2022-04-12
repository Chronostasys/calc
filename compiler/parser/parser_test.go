package parser

import (
	"testing"

	"github.com/llir/llvm/ir"
)

func TestParser_defineAndAssign(t *testing.T) {
	p := NewParser("main", "", ir.NewModule(), map[string]bool{})
	p.lexer.SetInput("a := struct{i int}{i:10}")
	_, err := p.defineAndAssign()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
