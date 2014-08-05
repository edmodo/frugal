package main

import (
	. "github.com/edmodo/frugal/parser"
)

type NameBinder struct {
	context *CompileContext
	tree    *ParseTree
}

// Find all name references in the parse tree and bind them to declarations.
func bindNames(context *CompileContext, tree *ParseTree) bool {
	binder := NameBinder{
		context: context,
		tree:    tree,
	}
	binder.bind()
	return !context.HasErrors()
}

// Walk top-level nodes.
func (this *NameBinder) bind() {
	for _, node := range this.tree.Nodes {
		this.bindNamesInNode(node)
	}
}

// Find unbound names in nodes and bind them.
func (this *NameBinder) bindNamesInNode(node Node) {
	switch node.(type) {
	case *TypedefNode:
		node := node.(*TypedefNode)
		this.bindType(node.Type)

	case *ConstNode:
		node := node.(*ConstNode)
		this.bindType(node.Type)
		this.bindNamesInNode(node.Init)

	case *ServiceNode:
		node := node.(*ServiceNode)
		if node.Extends != nil {
			this.bindNamesInNode(node.Extends)
		}
		for _, method := range node.Methods {
			this.bindType(method.ReturnType)
			for _, arg := range method.Args {
				this.bindType(arg.Type)
			}
			for _, arg := range method.Throws {
				this.bindType(arg.Type)
			}
		}

	case *NameProxyNode:
		node := node.(*NameProxyNode)
		node.Binding, node.Tail = this.resolvePath(node.Path)

	case *MapNode:
		node := node.(*MapNode)
		for _, entry := range node.Entries {
			this.bindNamesInNode(entry.Key)
			this.bindNamesInNode(entry.Value)
		}

	case *ListNode:
		node := node.(*ListNode)
		for _, expr := range node.Exprs {
			this.bindNamesInNode(expr)
		}

	case *StructNode:
		node := node.(*StructNode)
		for _, field := range node.Fields {
			this.bindType(field.Type)
			if field.Default != nil {
				this.bindNamesInNode(field.Default)
			}
		}
	}
}

// Resolve the named components of a type to a struct or typedef.
func (this *NameBinder) bindType(ttype Type) {
	switch ttype.(type) {
	case *NameProxyNode:
		ttype := ttype.(*NameProxyNode)
		ttype.Binding, ttype.Tail = this.resolvePath(ttype.Path)

	case *ListType:
		ttype := ttype.(*ListType)
		this.bindType(ttype.Inner)

	case *MapType:
		ttype := ttype.(*MapType)
		this.bindType(ttype.Key)
		this.bindType(ttype.Value)
	}
}

// Resolve the components of a path into a node and accessors. For example,
//   "types.crab", for a struct crab in types.thrift, will return:
//      node = crab, tail = []
//   "types.Flags.ADMIN", for an enum Flags in types.thrift, will return:
//      node = Flags, tail = [ADMIN]
//
// This information is passed to the type checking phase.
func (this *NameBinder) resolvePath(path []*Token) (Node, []*Token) {
	root := path[0]

	// Resolve to global symbols first.
	if _, ok := this.tree.Names[root.Identifier()]; ok {
		return this.resolvePathInPackage(path, this.tree)
	}

	// Otherwise, go to the package.
	if pkg, ok := this.tree.Includes[root.Identifier()]; ok {
		if len(path) == 1 {
			this.context.ReportError(root.Loc.Start, "name '%s' is a package", root.Identifier())
			return nil, nil
		}
		return this.resolvePathInPackage(path[1:], pkg)
	}

	// Lastly.. fail.
	this.context.ReportError(
		root.Loc.Start,
		"could not find any definition or package for name '%s'",
		root.Identifier(),
	)
	return nil, nil
}

func (this *NameBinder) resolvePathInPackage(path []*Token, tree *ParseTree) (Node, []*Token) {
	root := path[0]

	node, ok := tree.Names[root.Identifier()]
	if !ok {
		this.context.ReportError(
			root.Loc.Start,
			"name '%s' not found in package '%s'",
			root.Identifier(),
			tree.Package,
		)
		return nil, nil
	}

	return node, path[1:]
}
