package parser

import (
	"fmt"
)

// Base interface for all AST nodes.
type Node interface {
	Loc() Location
	NodeType() string
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
	return PrettyPrintMap[this.Tok.Kind]
}

// A named type must be resolved to a definition somewhere (for example, users.User).
type NamedType struct {
	Path []*Token
}

func (this *NamedType) String() string {
	return JoinIdentifiers(this.Path)
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

type EnumEntry struct {
	// Name token (always an identifier).
	Name *Token

	// Initializer (nil, or a TOK_LITERAL_INT).
	Value *Token

	// Constant value, filled in by semantic analysis.
	ConstVal int32
}

// Encapsulates an enum definition.
type EnumNode struct {
	Range   Location
	Name    *Token
	Entries []*EnumEntry

	// Map from name -> Entry. Filled in by semantic analysis.
	Names map[string]*EnumEntry
}

func NewEnumNode(loc Location, name *Token, fields []*EnumEntry) *EnumNode {
	return &EnumNode{
		Range:   loc,
		Name:    name,
		Entries: fields,
		Names:   map[string]*EnumEntry{},
	}
}

func (this *EnumNode) Loc() Location {
	return this.Range
}

func (this *EnumNode) NodeType() string {
	return "enum"
}

type StructField struct {
	// The token which contains the order number, or nil if not present.
	Order *Token

	// A token containing TOK_OPTIONAL or TOK_REQUIRED.
	Spec *Token

	// The type of the field.
	Type Type

	// The name of the field.
	Name *Token

	// The default value, or nil if not present.
	Default Node
}

// Encapsulates struct definition.
type StructNode struct {
	Range Location

	// Either TOK_EXCEPTION or TOK_STRUCT.
	Tok *Token

	// Struct/exception name and fields.
	Name   *Token
	Fields []*StructField

	// Map from name -> StructField. Filled in by semantic analysis.
	Names map[string]*StructField
}

func NewStructNode(loc Location, kind *Token, name *Token, fields []*StructField) *StructNode {
	return &StructNode{
		Range:  loc,
		Tok:    kind,
		Name:   name,
		Fields: fields,
		Names:  map[string]*StructField{},
	}
}

func (this *StructNode) NodeType() string {
	return PrettyPrintMap[this.Tok.Kind]
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

func (this *LiteralNode) NodeType() string {
	return "literal"
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

func (this *ListNode) NodeType() string {
	return "list"
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

func (this *MapNode) NodeType() string {
	return "map"
}

// Encapsulates a name or path of names.
type NameProxyNode struct {
	// Path components.
	Path []*Token

	// Node that this name binds to.
	Node Node
}

func NewNameProxyNode(path []*Token) *NameProxyNode {
	return &NameProxyNode{
		Path: path,
	}
}

func (this *NameProxyNode) NodeType() string {
	return "name"
}

func (this *NameProxyNode) Loc() Location {
	first := this.Path[0]
	last := this.Path[len(this.Path)-1]
	return Location{first.Loc.Start, last.Loc.End}
}

func (this *NameProxyNode) String() string {
	return JoinIdentifiers(this.Path)
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

	// Map of name -> argument. Filled in by semantic analysis.
	Names map[string]*ServiceMethodArg
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

func (this *ServiceNode) NodeType() string {
	return "service"
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

func (this *ConstNode) NodeType() string {
	return "constant"
}

// Encapsulates a typedef definition.
type TypedefNode struct {
	Range Location
	Type  Type
	Name  *Token
}

func (this *TypedefNode) Loc() Location {
	return this.Range
}

func (this *TypedefNode) NodeType() string {
	return "typedef"
}

type ParseTree struct {
	// Mapping of language -> namespace.
	Namespaces map[string]string

	// List of include paths.
	Includes map[string]*ParseTree

	// Root nodes in the syntax tree.
	Nodes []Node

	// The original file path.
	Path string

	// The package name this file would be imported, in thrift.
	Package string

	// Name to node mapping, filled in by semantic analysis.
	Names map[string]Node
}

func NewParseTree(file string) *ParseTree {
	return &ParseTree{
		Namespaces: map[string]string{},
		Includes:   map[string]*ParseTree{},
		Path:       file,
		Names:      map[string]Node{},
	}
}
