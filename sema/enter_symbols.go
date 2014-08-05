package main

import (
	. "github.com/edmodo/frugal/parser"
)

// Enter all symbols into their scopes.
func enterSymbols(context *CompileContext, tree *ParseTree) bool {
	for _, node := range tree.Nodes {
		enterSymbolsForNode(context, tree, node)
	}

	return !context.HasErrors()
}

func enterSymbolsForNode(context *CompileContext, tree *ParseTree, node Node) {
	switch node.(type) {
	case *EnumNode:
		node := node.(*EnumNode)
		enterGlobalSymbol(context, tree, node.Name, node)
		enterEnumSymbols(context, node)

	case *StructNode:
		node := node.(*StructNode)
		enterGlobalSymbol(context, tree, node.Name, node)
		enterStructSymbols(context, node)

	case *TypedefNode:
		node := node.(*TypedefNode)
		enterGlobalSymbol(context, tree, node.Name, node)

	case *ServiceNode:
		node := node.(*ServiceNode)
		enterGlobalSymbol(context, tree, node.Name, node)
		checkServiceSymbols(context, node)

	case *ConstNode:
		node := node.(*ConstNode)
		enterGlobalSymbol(context, tree, node.Name, node)
	}
}

func enterGlobalSymbol(context *CompileContext, tree *ParseTree, name *Token, node Node) {
	if prev, ok := tree.Names[name.Identifier()]; ok {
		context.ReportError(
			name.Loc.Start,
			"name '%s' was already declared as a %s on %s",
			name.Identifier(),
			prev.NodeType(),
			prev.Loc().Start,
		)
		return
	}

	tree.Names[name.Identifier()] = node
}

func enterEnumSymbols(context *CompileContext, node *EnumNode) {
	value := int32(0)

	for _, entry := range node.Entries {
		if entry.Value != nil {
			value = int32(entry.Value.IntLiteral())
			if int64(value) != entry.Value.IntLiteral() {
				context.ReportError(entry.Value.Loc.Start, "value does not fit in a 32-bit integer")
			}
		}

		entry.ConstVal = value
		value++

		name := entry.Name
		if prev, ok := node.Names[name.Identifier()]; ok {
			context.ReportRedeclaration(name.Loc.Start, prev.Name)
			continue
		}

		node.Names[name.Identifier()] = entry
	}
}

func enterStructSymbols(context *CompileContext, node *StructNode) {
	for _, field := range node.Fields {
		if prev, ok := node.Names[field.Name.Identifier()]; ok {
			context.ReportRedeclaration(field.Name.Loc.Start, prev.Name)
			continue
		}

		node.Names[field.Name.Identifier()] = field
	}
}

func checkServiceSymbols(context *CompileContext, node *ServiceNode) {
	symbols := map[string]*ServiceMethod{}

	// Methods must have unique names.
	for _, method := range node.Methods {
		name := method.Name
		if prev, ok := symbols[name.Identifier()]; ok {
			context.ReportRedeclaration(name.Loc.Start, prev.Name)
			continue
		}
		symbols[name.Identifier()] = method

		// Names must be unique across throws and args.
		argNames := map[string]*ServiceMethodArg{}
		for _, arg := range append(method.Args, method.Throws...) {
			if prev, ok := argNames[arg.Name.Identifier()]; ok {
				context.ReportRedeclaration(arg.Name.Loc.Start, prev.Name)
				continue
			}
			argNames[arg.Name.Identifier()] = arg
		}
	}
}
