package parser

import (
	"fmt"
	"io"
)

type AstPrinter struct {
	fp     io.Writer
	tree   *ParseTree
	prefix string
}

func (this *AstPrinter) fprintf(msg string, args ...interface{}) {
	text := this.prefix + fmt.Sprintf(msg, args...)
	_, err := this.fp.Write([]byte(text))
	if err != nil {
		panic(err)
	}
}

func (this *AstPrinter) indent() {
	this.prefix += "  "
}

func (this *AstPrinter) dedent() {
	this.prefix = this.prefix[:len(this.prefix) - 2]
}

func (this *AstPrinter) printArg(arg *ServiceMethodArg) {
	this.indent()
	msg := ""
	if arg.Order != nil {
		msg += fmt.Sprintf("%d: ", arg.Order.IntLiteral())
	}
	msg += fmt.Sprintf("%s ", arg.Type.String())
	msg += fmt.Sprintf("%s", arg.Name.Identifier())
	this.fprintf("%s\n", msg)
	this.dedent()
}

func (this *AstPrinter) printMethod(method *ServiceMethod) {
	extra := ""
	if method.OneWay != nil {
		extra = "oneway"
	}

	this.fprintf("[ method %s %s\n", method.Name.Identifier(), extra)
	this.indent()

	this.fprintf("args = \n")
	for _, arg := range method.Args {
		this.printArg(arg)
	}

	this.fprintf("throws = \n")
	for _, arg := range method.Throws {
		this.printArg(arg)
	}

	this.dedent()
}

func (this *AstPrinter) print() {
	for key, value := range this.tree.Namespaces {
		this.fprintf("namespace %s %s\n", key, value)
	}
	if len(this.tree.Namespaces) > 0 {
		this.fprintf("\n")
	}

	for include, _ := range this.tree.Includes {
		this.fprintf("include \"%s.thrift\"\n", include)
	}
	if len(this.tree.Includes) > 0 {
		this.fprintf("\n")
	}

	for _, node := range this.tree.Nodes {
		switch node.(type) {
		case *EnumNode:
			node := node.(*EnumNode)
			this.fprintf("[ enum %s\n", node.Name.Identifier())
			this.indent()
			for _, field := range node.Fields {
				this.fprintf("%s\n", field.Identifier())
			}
			this.dedent()

		case *StructNode:
			node := node.(*StructNode)
			this.fprintf("[ %s %s\n", PrettyPrintMap[node.Tok.Kind], node.Name.Identifier())
			this.indent()
			for _, field := range node.Fields {
				msg := ""
				if field.Order != nil {
					msg += fmt.Sprintf("%d: ", field.Order.IntLiteral())
				}
				msg += fmt.Sprintf("%s ", PrettyPrintMap[field.Spec.Kind])
				msg += fmt.Sprintf("%s", field.Name.Identifier())
				this.fprintf("%s\n", msg)
			}
			this.dedent()

		case *ServiceNode:
			node := node.(*ServiceNode)
			header := fmt.Sprintf("[ service %s", node.Name.Identifier())
			if node.Extends != nil {
				header += fmt.Sprintf(" extends %s", node.Extends.String())
			}
			this.fprintf("%s\n", header)
			this.indent()
			for _, method := range node.Methods {
				this.printMethod(method)
			}
			this.dedent()

		default:
			this.fprintf("Unrecognized node! %T %v\n", node, node)
		}
	}
}

func (this *ParseTree) Print(fp io.Writer) {
	printer := AstPrinter{
		fp: fp,
		tree: this,
	}
	printer.print()
}
