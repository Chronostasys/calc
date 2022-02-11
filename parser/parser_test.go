package parser

import (
	"testing"
)

func TestParser_defineAndAssign(t *testing.T) {
	p := NewParser()
	p.lexer.SetInput("a := struct{i int}{i:10}")
	_, err := p.defineAndAssign()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
