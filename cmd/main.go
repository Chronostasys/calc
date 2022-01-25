package main

import (
	"bufio"
	"os"

	"github.com/Chronostasys/calculator_go/parser"
)

func main() {
	r := bufio.NewReader(os.Stdin)
	for {
		bs, _, err := r.ReadLine()
		if err != nil {
			println(err.Error())
			return
		}
		re := parser.Parse(string(bs))
		println("=>", re)
	}
}
