package main

import (
	"flag"
	"log"
	"os"

	"github.com/Chronostasys/calculator_go/parser"
)

func main() {
	var outf string
	flag.StringVar(&outf, "o", "out.ll", "llvm ir file")
	flag.Parse()
	m := parser.ParseCurentDir()
	f, err := os.OpenFile(outf, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalln(err)
	}
	err = f.Truncate(0)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	_, err = m.WriteTo(f)
	if err != nil {
		log.Fatalln(err)
	}
	err = f.Sync()
	if err != nil {
		log.Fatalln(err)
	}
}
