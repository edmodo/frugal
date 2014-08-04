package main

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func main() {
	context := NewCompileContext()

	files, err := ioutil.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		filename := file.Name()
		if !strings.Contains(filename, ".thrift") {
			continue
		}

		(func() {
			context.Enter(filename)
			defer context.Leave()
			parser, _ := NewParser(context)

			tree := parser.Parse()
			if tree == nil {
				context.PrintErrors()
				return
			}
			fmt.Printf("%v\n", tree)
		})()
	}
}
