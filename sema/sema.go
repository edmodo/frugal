// vim: set ts=4 sw=4 tw=99 noet:
//
// Copyright 2014, Edmodo, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this work except in compliance with the License.
// You may obtain a copy of the License in the LICENSE file, or at:
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language
// governing permissions and limitations under the License.

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
