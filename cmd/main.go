package main

import (
	"io/ioutil"

	"github.com/Chronostasys/calculator_go/parser"
)

func main() {
	bs, _ := ioutil.ReadFile("test.calc")
	code := string(bs)
	ir := parser.Parse(code)
	ioutil.WriteFile("test.ll", []byte(ir), 0777)
}
