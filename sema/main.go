package main

import (
	"flag"
	"fmt"
	"os"

	parser "github.com/edmodo/frugal/parser"
)

var (
	dumpFlag = flag.Bool("dump", false, "Dump the AST and exit.")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: file1 file2 file3...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	for _, file := range flag.Args() {
		context := parser.NewCompileContext()
		tree := context.ParseRecursive(file)
		if tree == nil {
			context.PrintErrors()
			os.Exit(1)
		}

		if *dumpFlag {
			tree.Print(os.Stdout)
		}
	}
}
