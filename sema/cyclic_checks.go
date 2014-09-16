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

type CyclicChecker struct {
	context *CompileContext
	tree    *ParseTree
}

func cyclicCheck(context *CompileContext, tree *ParseTree) bool {
	checker := &CyclicChecker{
		context: context,
		tree:    tree,
	}
	return checker.check()
}

func (this *CyclicChecker) check() bool {
	for _, node := range this.tree.Nodes {
		switch node.(type) {
		case *StructNode:
			this.checkCyclicStruct(node.(*StructNode))

		case *ServiceNode:
			this.checkCyclicService(node.(*ServiceNode))
		}
	}
	return !this.context.HasErrors()
}

func (this *CyclicChecker) findNestedType(ttype Type, target *StructNode) bool {
	// Peel away typedefs.
	ttype, binding := ttype.Resolve()

	switch ttype.(type) {
	case *ListType:
		ttype := ttype.(*ListType)
		return this.findNestedType(ttype.Inner, target)

	case *MapType:
		ttype := ttype.(*MapType)
		if this.findNestedType(ttype.Key, target) || this.findNestedType(ttype.Value, target) {
			return true
		}

	case *NameProxyNode:
		node, ok := binding.(*StructNode)
		if !ok {
			// Not a struct, so it can't be cyclic.
			return false
		}

		if node == target {
			return true
		}

		// Search the struct's fields
		for _, field := range node.Fields {
			if this.findNestedType(field.Type, target) {
				return true
			}
		}
	}

	return false
}

func (this *CyclicChecker) checkCyclicStruct(node *StructNode) {
	// For each field type, recursively traverse compound types to find references
	// to the outer struct. This algorithm is not very intelligent - for example -
	// it will not cache types it has already seen.
	for _, field := range node.Fields {
		if this.findNestedType(field.Type, node) {
			this.context.ReportError(
				field.Name.Loc.Start,
				"field '%s' introduces a cyclic reference to struct '%s'",
				field.Name.Identifier(),
				node.Name.Identifier(),
			)
			break
		}
	}
}

func (this *CyclicChecker) checkCyclicService(node *ServiceNode) {
	if node.Extends == nil {
		return
	}

	parent := node.Extends.Binding.(*ServiceNode)
	for {
		if parent == node {
			this.context.ReportError(node.Extends.Loc().Start, "service extension is cyclic or extends from itself")
			return
		}

		if parent.Extends == nil {
			// Chain stops - exit with no error.
			return
		}

		parent = parent.Extends.Binding.(*ServiceNode)
	}
}
