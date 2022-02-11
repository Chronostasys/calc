package main

import (
	"flag"
	"log"
	"os"

	"github.com/Chronostasys/calc/compiler/parser"
)

func main() {
	var indir, outf string
	flag.StringVar(&indir, "d", "../test", "source repo dir")
	flag.StringVar(&outf, "o", "out.ll", "llvm ir file")
	flag.Parse()
	m := parser.ParseDir(indir)
	f, err := os.Create(outf)
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
