package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Chronostasys/calc/compiler/parser"
)

func main() {
	var indir, outf string
	flag.StringVar(&indir, "d", ".", "source repo dir")
	flag.StringVar(&outf, "o", "out.ll", "llvm ir file")
	flag.Parse()
	since := time.Now()
	defer func() {
		err := recover()
		if err != nil {
			panic(err)
		}
		fmt.Printf("	compile secceed. output file: %s\n", outf)
		fmt.Printf("	time eplased: %v\n", time.Since(since))
	}()
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
