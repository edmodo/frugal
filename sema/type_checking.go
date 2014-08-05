package main

import (
	. "github.com/edmodo/frugal/parser"
)

type TypeChecker struct {
	context *CompileContext
	tree    *ParseTree
}

func typeCheck(context *CompileContext, tree *ParseTree) bool {
	checker := TypeChecker{
		context: context,
		tree: tree,
	}
	return checker.check()
}

func (this *TypeChecker) affirmNodeIsType(node Node) bool {
	switch node.(type) {
	case *StructNode, *TypedefNode:
		return true

	default:
		return false
	}
}

// Check that the given type actually maps to something defining a type.
func (this *TypeChecker) affirmType(ttype Type) bool {
	switch ttype.(type) {
	case *NameProxyNode:
		node := ttype.(*NameProxyNode)
		if this.affirmNodeIsType(node.Binding) {
			if len(node.Tail) == 0 {
				return true
			}
		}

		// Either the name does not resolve to a type, or it does, but other
		// stuff comes after it (like a struct or enum field).
		this.context.ReportError(
			node.Loc().Start,
			"expected a type, but '%s' does not resolve to a type",
			node.String(),
		)
		return false

	case *ListType:
		ttype := ttype.(*ListType)
		return this.affirmType(ttype.Inner)

	case *MapType:
		ttype := ttype.(*MapType)
		if !this.affirmType(ttype.Key) {
			return false
		}
		return this.affirmType(ttype.Value)
	}

	panic("unexpected node kind")
	return false
}

func (this *TypeChecker) checkType(ttype Type, value Node) {
}

func (this *TypeChecker) checkStruct(node *StructNode) {
	for _, field := range node.Fields {
		// :TODO: check order
		this.affirmType(field.Type)
		if field.Default != nil {
			this.checkType(field.Type, field.Default)
		}
	}
}

func (this *TypeChecker) checkService(node *ServiceNode) {
	if node.Extends != nil {
		// The inherited node must be a service node.
		_, isService := node.Extends.Binding.(*ServiceNode)
		if !isService || len(node.Extends.Tail) > 0 {
			// Either the node is not a service node, or there are extra
			// components in its path.
			this.context.ReportError(
				node.Loc().Start,
				"name '%s' must be a service definition",
				node.Extends.String(),
			)
		}
	}

	for _, method := range node.Methods {
		this.affirmType(method.ReturnType)

		for _, arg := range method.Args {
			this.affirmType(arg.Type)
		}
		for _, throws := range method.Throws {
			if !this.affirmType(throws.Type) {
				continue
			}

			// Exceptions can be used as structs anywhere, but in particular,
			// a 'throws' clause can only use exceptions.
			ttype, binding := throws.Type.Resolve()
			if binding == nil {
				this.context.ReportError(
					ttype.Loc().Start,
					"expected an exception, but got type '%s'",
					ttype.String(),
				)
			}
		}
	}
}

func (this *TypeChecker) checkNode(node Node) {
	// We only need to look at top-level node kinds here.
	switch node.(type) {
	case *ConstNode:
		node := node.(*ConstNode)
		this.affirmType(node.Type)

	case *StructNode:
		node := node.(*StructNode)
		this.checkStruct(node)

	case *ServiceNode:
		node := node.(*ServiceNode)
		this.checkService(node)
	}
}

func (this *TypeChecker) check() bool {
	for _, node := range this.tree.Nodes {
		this.checkNode(node)
	}
	return !this.context.HasErrors()
}
