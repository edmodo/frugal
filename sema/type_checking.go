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
		tree:    tree,
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
		node.Init = this.checkType(node.Type, node.Init)

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
		// We're referencing a defined type by name. Make sure it's actually a type.
		node := ttype.(*NameProxyNode)
		if this.isNodeATypeDeclaration(node.Binding) {
			// tail > 0 would indicate that there are extra components in the path.
			// thrift IDL has no way to nest type names (i.e. you cannot declare
			// types inside a struct).
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
		if this.affirmType(ttype.Key) && this.affirmType(ttype.Value) {
			return true
		}
		return false
	}

	panic("unexpected node kind")
	return false
}

func (this *TypeChecker) isNodeATypeDeclaration(node Node) bool {
	switch node.(type) {
	case *StructNode, *TypedefNode, *EnumNode:
		return true

	default:
		return false
	}
}

// Check that the value node can be assigned to the given type.
func (this *TypeChecker) checkType(ttype Type, value Node) *ValueNode {
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

		if enum, ok := ttype.Binding.(*EnumNode); ok {
			return this.checkEnumType(enum, value)
		}
		if tstruct, ok := ttype.Binding.(*StructNode); ok {
			return this.checkStructType(tstruct, value)
		}
		this.context.ReportError(ttype.Loc().Start, "cannot use type '%s' here", ttype.String())
		return nil
	}

	panic("unexpected type")
}

// Checks whether a literal integer can be coerced to a 32-bit integer.
func (this *TypeChecker) toI32(lit *Token) (int32, bool) {
	value := int32(lit.IntLiteral())
	if int64(value) == lit.IntLiteral() {
		return value, true
	}
	this.context.ReportError(
		lit.Loc.Start,
		"value '%d' does not fit in a 32-bit integer",
		lit.IntLiteral(),
	)
	return 0, false
}

// Check assignment of a value to a builtin type.
func (this *TypeChecker) checkBuiltinType(ttype *BuiltinType, value Node) *ValueNode {
	lit, ok := value.(*LiteralNode)
	if !ok {
		this.context.ReportError(value.Loc().Start, "cannot coerce '%s' to type '%s'", value.NodeType(), ttype.String())
		return nil
	}

	switch ttype.Tok.Kind {
	case TOK_BOOL:
		if lit.Lit.Kind == TOK_TRUE {
			return &ValueNode{value, TOK_BOOL, true}
		}
		if lit.Lit.Kind == TOK_FALSE {
			return &ValueNode{value, TOK_BOOL, false}
		}
	case TOK_I32:
		if lit.Lit.Kind == TOK_LITERAL_INT {
			i32, ok := this.toI32(lit.Lit)
			if !ok {
				return nil
			}
			return &ValueNode{value, TOK_I32, i32}
		}
	case TOK_I64:
		if lit.Lit.Kind == TOK_LITERAL_INT {
			return &ValueNode{value, TOK_I64, lit.Lit.IntLiteral()}
		}
	case TOK_STRING:
		if lit.Lit.Kind == TOK_LITERAL_STRING {
			return &ValueNode{value, TOK_STRING, lit.Lit.StringLiteral()}
		}
	}

	this.context.ReportError(
		lit.Loc().Start,
		"cannot coerce type '%s' to type '%s'",
		lit.TypeString(),
		ttype.String(),
	)
	return nil
}

// Check assignment of a value to a list type.
func (this *TypeChecker) checkListType(ttype *ListType, value Node) *ValueNode {
	list, ok := value.(*ListNode)
	if !ok {
		this.context.ReportError(value.Loc().Start, "cannot coerce '%s' to a list", value.NodeType())
		return nil
	}

	for _, expr := range list.Exprs {
		value := this.checkType(ttype.Inner, expr)
		if value == nil {
			return nil
		}
		list.Values = append(list.Values, value)
	}

	// Wrap the list into a value node.
	return &ValueNode{list, TOK_LIST, list}
}

func (this *TypeChecker) checkMapType(ttype *MapType, value Node) *ValueNode {
	tmap, ok := value.(*MapNode)
	if !ok {
		this.context.ReportError(value.Loc().Start, "cannot coerce '%s' to a map", value.NodeType())
		return nil
	}

	// Check and resolve each key/value in the map.
	for _, entry := range tmap.Entries {
		keyVal := this.checkType(ttype.Key, entry.Key)
		if keyVal == nil {
			return nil
		}
		valueVal := this.checkType(ttype.Value, entry.Value)
		if valueVal == nil {
			return nil
		}
		entry.KeyVal = keyVal
		entry.ValueVal = valueVal
	}

	// Wrap the map into a value node.
	return &ValueNode{tmap, TOK_MAP, tmap}
}

func (this *TypeChecker) checkStructType(tstruct *StructNode, inValue Node) *ValueNode {
	// This uses the same initialization syntax as maps.
	value, ok := inValue.(*MapNode)
	if !ok {
		this.context.ReportError(inValue.Loc().Start, "value should be a struct initializer")
		return nil
	}

	init := StructInitializer{}
	for _, entry := range value.Entries {
		lit, ok := entry.Key.(*LiteralNode)
		if !ok || lit.Lit.Kind != TOK_LITERAL_STRING {
			this.context.ReportError(entry.Key.Loc().Start, "expected a string literal with a struct field")
			return nil
		}

		fieldName := lit.Lit.StringLiteral()
		field, ok := tstruct.Names[fieldName]
		if !ok {
			this.context.ReportError(
				entry.Key.Loc().Start,
				"field '%s' not found in struct '%s'",
				fieldName,
				tstruct.Name.Identifier(),
			)
			return nil
		}

		newval := this.checkType(field.Type, entry.Value)
		if newval == nil {
			return nil
		}
		init[field] = newval
	}

	// Warn about any required fields that are not present.
	for _, field := range tstruct.Fields {
		if field.Spec != nil && field.Spec.Kind != TOK_REQUIRED {
			continue
		}
		if field.Default != nil {
			continue
		}

		if _, ok := init[field]; !ok {
			this.context.ReportError(
				value.Loc().Start,
				"required field '%s' in struct '%s' is not initialized",
				field.Name.Identifier(),
				tstruct.Name.Identifier(),
			)
		}
	}

	return &ValueNode{value, TOK_STRUCT, init}
}

func (this *TypeChecker) checkEnumType(enum *EnumNode, inValue Node) *ValueNode {
	// We're assigning to an enum; only a name referencing an enum field can do that.
	value, ok := inValue.(*NameProxyNode)
	if !ok {
		this.context.ReportError(inValue.Loc().Start, "value is not a member of enum '%s'", enum.Name.Identifier())
		return nil
	}

	other, ok := value.Binding.(*EnumNode)
	if !ok {
		this.context.ReportError(value.Loc().Start, "value is not a member of enum '%s'", enum.Name.Identifier())
		return nil
	}

	// Check that we're not trying to use an enum definition as a value.
	if len(value.Tail) == 0 {
		this.context.ReportError(
			value.Loc().Start,
			"cannot use enum '%s' as a value",
			other.Name.Identifier(),
		)
		return nil
	}

	// Check that we're not trying to access members of enum fields.
	if len(value.Tail) > 1 {
		this.context.ReportError(
			value.Loc().Start,
			"%s is not a member of enum '%s'",
			JoinIdentifiers(value.Tail),
			other.Name.Identifier(),
		)
		return nil
	}

	// Look up the final path component in the enum.
	name := value.Tail[0]
	entry, ok := other.Names[name.Identifier()]
	if !ok {
		this.context.ReportError(
			name.Loc.Start,
			"%s is not a member of enum '%s'",
			name,
			other.Name.Identifier(),
		)
		return nil
	}

	// Make sure it's the same enum. We do this after we fetch the field, in order
	// to report any name access errors.
	if other != enum {
		this.context.ReportError(
			value.Loc().Start,
			"cannot coerce enum '%s' to enum '%s'",
			other.Name.Identifier(),
			enum.Name.Identifier(),
		)
		return nil
	}

	return &ValueNode{value, TOK_ENUM, entry}
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
	orders := map[int32]*StructField{}

	for _, field := range node.Fields {
		if field.Order == nil {
			// Upstream thrift has this as a warning. That seems pointless, so we error.
			this.context.ReportError(
				field.Name.Loc.Start,
				"field '%s' should have an explicit order, for better compatibility",
				field.Name.Identifier(),
			)
		} else {
			// Check that the order number has not already been seen.
			if order, ok := this.toI32(field.Order); ok {
				if prev, ok := orders[order]; ok {
					this.context.ReportError(
						field.Order.Loc.Start,
						"field '%s' has the same ordering as field '%s'",
						field.Name.Identifier(),
						prev.Name.Identifier(),
					)
				} else {
					orders[order] = field
				}

				// The order cannot be a negative number.
				if order <= 0 {
					this.context.ReportError(
						field.Order.Loc.Start,
						"field '%s' must be an integer greater than 0",
						field.Name.Identifier(),
					)
				}
			}
		}

		this.affirmType(field.Type)
		this.checkNotVoid(field.Type)

		if field.Default != nil {
			field.Default = this.checkType(field.Type, field.Default)
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
		this.checkOrdering(method.Args, "argument")

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
		this.checkOrdering(method.Throws, "exception")
	}
}

func (this *TypeChecker) checkOrdering(args []*ServiceMethodArg, kind string) {
	orders := map[int32]*ServiceMethodArg{}

	for _, arg := range args {
		if arg.Order == nil {
			// Upstream thrift has this as a warning. That seems pointless, so we error.
			this.context.ReportError(
				arg.Name.Loc.Start,
				"%s '%s' should have an explicit order, for better compatibility",
				kind,
				arg.Name.Identifier(),
			)
			continue
		}

		// Check that the order number has not already been seen.
		order, ok := this.toI32(arg.Order)
		if !ok {
			continue
		}

		if prev, ok := orders[order]; ok {
			this.context.ReportError(
				arg.Order.Loc.Start,
				"%s '%s' has the same ordering as %s '%s'",
				kind,
				arg.Name.Identifier(),
				kind,
				prev.Name.Identifier(),
			)
		} else {
			orders[order] = arg
		}

		// The order cannot be a negative number.
		if order <= 0 {
			this.context.ReportError(
				arg.Order.Loc.Start,
				"%s '%s' must be an integer greater than 0",
				kind,
				arg.Name.Identifier(),
			)
		}
	}
}
