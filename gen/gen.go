package gen

import (
	"bytes"
	"fmt"
)

type Generator struct {
	buffer bytes.Buffer
	indent string
	prefix string

	// Insert spaces instead of hard tabs.
	softTabs bool

	// If using spaces, this controls the indent size.
	indentSize int
}

// Creates a new generator, defaulting to hard tabs.
func NewGenerator() *Generator {
	return &Generator{
		indent:     "\t",
		prefix:     "",
		softTabs:   false,
		indentSize: 1,
	}
}

// Set soft tabs. This should be called after calling NewGenerator, before any
// code has been generated.
func (this *Generator) SetSoftTabs(softTabs bool, indentSize int) {
	this.softTabs = softTabs
	this.indentSize = indentSize

	indent := " "
	for i := 1; i <= indentSize; i++ {
		indent += " "
	}

	this.indent = indent
	this.prefix = ""
}

// Returns the current indent prefix string.
func (this *Generator) Prefix() string {
	return this.indent
}

// Emits a prefix string.
func (this *Generator) EmitPrefix() {
	this.Emit(this.prefix)
}

// Adds an indent.
func (this *Generator) Indent() {
	this.prefix += this.indent
}

// Removes an indent.
func (this *Generator) Dedent() {
	this.prefix = this.prefix[:len(this.prefix)-this.indentSize]
}

// Writes a raw string to the output buffer.
func (this *Generator) Emit(str string) {
	this.buffer.WriteString(str)
}

// Write a line, emitting an indent prefix beforehand.
func (this *Generator) Writeln(sfmt string, args ...interface{}) {
	this.EmitPrefix()
	this.Emit(fmt.Sprintf(sfmt, args...))
	this.Emit("\n")
}

func (this *Generator) String() string {
	return string(this.buffer.Bytes())
}
