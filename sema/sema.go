package sema

import (
	. "github.com/edmodo/frugal/parser"
)

type PhaseCallback func(context *CompileContext, tree *ParseTree) bool

var compilePhases = []PhaseCallback{
	enterSymbols,
	bindNames,
	typeCheck,
	cyclicCheck,
	checkUnused,
}

func runPhase(context *CompileContext, phase PhaseCallback, tree *ParseTree) bool {
	context.Enter(tree.Path)
	defer context.Leave()

	return phase(context, tree)
}

func runPhases(context *CompileContext, trees []*ParseTree) bool {
	for _, phase := range compilePhases {
		for _, tree := range trees {
			if !runPhase(context, phase, tree) {
				return false
			}
		}
	}
	return true
}

func Analyze(context *CompileContext, tree *ParseTree) bool {
	trees := FlattenTrees(tree)
	return runPhases(context, trees)
}
