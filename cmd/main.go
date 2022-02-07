package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/Chronostasys/calculator_go/parser"
)

func main() {
	var inf, outf string
	flag.StringVar(&inf, "c", "test.calc", "source file")
	flag.StringVar(&outf, "o", "test.ll", "llvm ir file")
	flag.Parse()
	bs, err := ioutil.ReadFile(inf)
	if err != nil {
		log.Fatalln(err)
	}
	code := string(bs)
	ir := parser.Parse(code)
	err = ioutil.WriteFile(outf, []byte(ir), 0777)
	if err != nil {
		log.Fatalln(err)
	}
	// ast.PrintTable()
	// ast := parser.ParseAST(code)
	// fmt.Println(ast)
}
