package lexer

import (
	"fmt"
	"testing"
)

func TestScan(t *testing.T) {
	input = "120+34 + 089"
	for {
		code, val, eos := Scan()
		if eos {
			break
		}
		fmt.Println(code, val, eos)
	}
}
