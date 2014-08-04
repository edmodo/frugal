package parser

import (
	"fmt"
)

// Base interface for all AST nodes.
type Node interface {
	Loc() Location
}

// Base interface for all constructs that are parsed as a type expression.
type Type interface {
	String() string
}

// A builtin type is just a single token (such as i32).
type BuiltinType struct {
	Tok *Token
}

func (this *BuiltinType) String() string {
	return this.Tok.String()
}

// A named type must be resolved to a definition somewhere (for example, users.User).
type NamedType struct {
	Path []*Token
}

func (this *NamedType) String() string {
	return fmt.Sprintf("%v", this.Path)
}

// A list type is list<type>.
type ListType struct {
	Inner Type
}

func (this *ListType) String() string {
	return fmt.Sprintf("list<%s>", this.Inner.String())
}

// A map type is map<key, value>.
type MapType struct {
	Key   Type
	Value Type
}

func (this *MapType) String() string {
	return fmt.Sprintf("map<%s,%s>", this.Key.String(), this.Value.String())
}

// Encapsulates an enum definition.
type EnumNode struct {
	Range  Location
	Name   *Token
	Fields []*Token
}

func (this *EnumNode) Loc() Location {
	return this.Range
}

type StructField struct {
	// The token which contains the order number, or nil if not present.
	Order *Token

	// A token containing TOK_OPTIONAL or TOK_REQUIRED.
	Spec *Token

	// The name of the field.
	Name *Token

	// The default value, or nil if not present.
	Default Node
}

// Encapsulates struct definition.
type StructNode struct {
	Range  Location
	Name   *Token
	Fields []*StructField
}

func (this *StructNode) Loc() Location {
	return this.Range
}

// Encapsulates a literal.
type LiteralNode struct {
	// The token is either a TOK_LITERAL_INT or TOK_LITERAL_STRING.
	Lit *Token
}

func (this *LiteralNode) Loc() Location {
	return this.Lit.Loc
}

// A sequence of expressions.
type ListNode struct {
	Exprs []Node
}

func (this *ListNode) Loc() Location {
	return Location{
		Start: this.Exprs[0].Loc().Start,
		End:   this.Exprs[len(this.Exprs)-1].Loc().End,
	}
}

// A sequence of expression pairs, in a key-value mapping.
type MapNodeEntry struct {
	Key   Node
	Value Node
}
type MapNode struct {
	Entries []MapNodeEntry
}

func (this *MapNode) Loc() Location {
	return Location{
		Start: this.Entries[0].Key.Loc().Start,
		End:   this.Entries[len(this.Entries)-1].Value.Loc().End,
	}
}

// Encapsulates a name or path of names.
type NameProxyNode struct {
	Path []*Token
}

func (this *NameProxyNode) Loc() Location {
	first := this.Path[0]
	last := this.Path[len(this.Path)-1]
	return Location{first.Loc.Start, last.Loc.End}
}

type ServiceMethodArg struct {
	// The order of the argument, if present, as a TOK_LITERAL_INT
	Order *Token

	// The type expression of the argument.
	Type Type

	// The token containing the argument name.
	Name *Token
}

type ServiceMethod struct {
	// If non-nil, specifies that the method is one-way.
	OneWay *Token

	// The return type expression of the method.
	ReturnType Type

	// The name of the method.
	Name *Token

	// The argument list of the method.
	Args []*ServiceMethodArg

	// The list of throwable errors of the method.
	Throws []*ServiceMethodArg
}

// Encapsulates a service definition.
type ServiceNode struct {
	Range   Location
	Name    *Token
	Extends *NameProxyNode
	Methods []*ServiceMethod
}

func (this *ServiceNode) Loc() Location {
	return this.Range
}

// Encapsulates a constant variable definition.
type ConstNode struct {
	Range Location

	// The type of the constant variable.
	Type Type

	// The name of the constant variable.
	Name *Token

	// The initialization value of the constant variable.
	// This is always one of:
	//   LiteralNode
	//   ListNode
	//   MapNode
	Init Node
}

func (this *ConstNode) Loc() Location {
	return this.Range
}

type ParseTree struct {
	// Mapping of language -> namespace.
	Namespaces map[string]string

	// List of include paths.
	Includes map[string]*ParseTree

	// Root nodes in the syntax tree.
	Nodes []Node

	// The package name this file would be imported, in thrift.
	Package string
}

func NewParseTree() *ParseTree {
	return &ParseTree{
		Namespaces: map[string]string{},
		Includes:   map[string]*ParseTree{},
	}
}
