package main

import (
	. "github.com/edmodo/frugal/parser"
)

type PhaseCallback func(context *CompileContext, tree *ParseTree) bool

var compilePhases = []PhaseCallback{
	enterSymbols,
	bindNames,
}

func runPhases(context *CompileContext, trees []*ParseTree) bool {
	for _, phase := range compilePhases {
		for _, tree := range trees {
			if !phase(context, tree) {
				return false
			}
		}
	}
	return true
}
