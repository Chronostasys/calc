package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/Chronostasys/calculator_go/parser"
)

func main() {
	var outf string
	flag.StringVar(&outf, "o", "out.ll", "llvm ir file")
	flag.Parse()
	s := parser.ParseCurentDir()
	err := ioutil.WriteFile(outf, []byte(s), 0777)
	if err != nil {
		log.Fatalln(err)
	}
}
