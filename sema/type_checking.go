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

func (this *TypeChecker) checkNode(node Node) {
	// We only need to look at top-level node kinds here.
	switch node.(type) {
	case *TypedefNode:
		node := node.(*TypedefNode)
		this.affirmType(node.Type)

	case *ConstNode:
		node := node.(*ConstNode)
		this.affirmType(node.Type)
		this.checkType(node.Type, node.Init)

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

// Check that the given type actually maps to something defining a type.
func (this *TypeChecker) affirmType(ttype Type) bool {
	switch ttype.(type) {
	case *BuiltinType:
		return true

	case *NameProxyNode:
		node := ttype.(*NameProxyNode)
		if this.checkNodeDeclaresType(node.Binding) {
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

func (this *TypeChecker) checkNodeDeclaresType(node Node) bool {
	switch node.(type) {
	case *StructNode, *TypedefNode:
		return true

	default:
		return false
	}
}

// Check that the value node can be assigned to the given type.
func (this *TypeChecker) checkType(ttype Type, value Node) bool {
	// Reach past any typedefs.
	ttype, _ = ttype.Resolve()

	switch ttype.(type) {
	case *BuiltinType:
		ttype := ttype.(*BuiltinType)
		return this.checkBuiltinType(ttype, value)

	case *ListType:
		ttype := ttype.(*ListType)
		return this.checkListType(ttype, value)

	case *MapType:
		ttype := ttype.(*MapType)
		return this.checkMapType(ttype, value)

	case *NameProxyNode:
		ttype := ttype.(*NameProxyNode)
		this.context.ReportError(
			ttype.Loc().Start,
			"cannot use type '%s' here",
			ttype.String(),
		)
		return false
	}

	panic("unexpected type")
	return false
}

// Checks whether a literal integer can be coerced to a 32-bit integer.
func (this *TypeChecker) toI32(lit *LiteralNode) (int32, bool) {
	value := int32(lit.Lit.IntLiteral())
	if int64(value) == lit.Lit.IntLiteral() {
		return value, true
	}
	this.context.ReportError(
		lit.Loc().Start,
		"value '%d' does not fit in a 32-bit integer",
		lit.Lit.IntLiteral(),
	)
	return 0, false
}

// Check assignment of a value to a builtin type.
func (this *TypeChecker) checkBuiltinType(ttype *BuiltinType, value Node) bool {
	lit, ok := value.(*LiteralNode)
	if !ok {
		this.context.ReportError(
			value.Loc().Start,
			"cannot coerce '%s' to type '%s'",
			value.NodeType(),
			ttype.String(),
		)
		return false
	}

	switch ttype.Tok.Kind {
	case TOK_BOOL:
		if lit.Lit.Kind == TOK_TRUE || lit.Lit.Kind == TOK_FALSE {
			return true
		}
	case TOK_I32:
		if lit.Lit.Kind == TOK_LITERAL_INT {
			_, ok := this.toI32(lit)
			return ok
		}
	case TOK_I64:
		if lit.Lit.Kind == TOK_LITERAL_INT {
			return true
		}
	case TOK_STRING:
		if lit.Lit.Kind == TOK_LITERAL_STRING {
			return true
		}
	}

	this.context.ReportError(
		lit.Loc().Start,
		"cannot coerce type '%s' to type '%s'",
		lit.TypeString(),
		ttype.String(),
	)
	return false
}

// Check assignment of a value to a list type.
func (this *TypeChecker) checkListType(ttype *ListType, value Node) bool {
	list, ok := value.(*ListNode)
	if !ok {
		this.context.ReportError(
			value.Loc().Start,
			"cannot coerce '%s' to a list",
			value.NodeType(),
		)
		return false
	}

	for _, expr := range list.Exprs {
		if !this.checkType(ttype.Inner, expr) {
			return false
		}
	}

	return true
}

func (this *TypeChecker) checkMapType(ttype *MapType, value Node) bool {
	list, ok := value.(*MapNode)
	if !ok {
		this.context.ReportError(
			value.Loc().Start,
			"cannot coerce '%s' to a map",
			value.NodeType(),
		)
		return false
	}

	for _, entry := range list.Entries {
		if !this.checkType(ttype.Key, entry.Key) {
			return false
		}
		if !this.checkType(ttype.Value, entry.Value) {
			return false
		}
	}

	return true
}

// Check that a type is not void.
func (this *TypeChecker) checkNotVoid(ttype Type) {
	ttype, _ = ttype.Resolve()
	builtin, ok := ttype.(*BuiltinType)
	if !ok {
		return
	}

	if builtin.Tok.Kind == TOK_VOID {
		this.context.ReportError(ttype.Loc().Start, "void can only be used as a return type")
	}
}

func (this *TypeChecker) checkStruct(node *StructNode) {
	for _, field := range node.Fields {
		// :TODO: check order
		this.affirmType(field.Type)
		this.checkNotVoid(field.Type)

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
			// Either the node is not a service node, or there are extra components in its path.
			this.context.ReportError(
				node.Loc().Start,
				"name '%s' must be a service definition",
				node.Extends.String(),
			)
		}
	}

	// Check method signatures.
	for _, method := range node.Methods {
		this.affirmType(method.ReturnType)

		// Check argument types.
		for _, arg := range method.Args {
			this.affirmType(arg.Type)
			this.checkNotVoid(arg.Type)
		}

		// Check exception types.
		for _, throws := range method.Throws {
			if !this.affirmType(throws.Type) {
				continue
			}

			// Exceptions can be used as general structs, but a 'throws' clause can
			// only use exceptions.
			ttype, binding := throws.Type.Resolve()
			if binding == nil {
				this.context.ReportError(
					ttype.Loc().Start,
					"expected an exception, but got type '%s'",
					ttype.String(),
				)
				continue
			}

			node, ok := binding.(*StructNode)
			if !ok || node.Tok.Kind != TOK_EXCEPTION {
				this.context.ReportError(
					ttype.Loc().Start,
					"expected an exception, but got a %s",
					binding.NodeType(),
				)
				continue
			}
		}
	}
}
