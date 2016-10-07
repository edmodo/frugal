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

// `gen` provides a simple helper for writing text code generators. It buffers output (and can maintain separate buffers), and automatically tracks indenting. It is not a code generator itself.
package gen

import (
	"bytes"
	"fmt"
	"io"
)

type Generator struct {
	buffer *bytes.Buffer
	indent string
	prefix string

	// Insert spaces instead of hard tabs.
	softTabs bool

	// If using spaces, this controls the indent size.
	indentSize int

	// Use a separate buffer for the header, so codegen can track what needs to
	// be imported.
	body   *bytes.Buffer
	header *bytes.Buffer
}

// Creates a new generator, defaulting to hard tabs.
func NewGenerator() *Generator {
	body := new(bytes.Buffer)
	header := new(bytes.Buffer)
	return &Generator{
		buffer:     body,
		indent:     "\t",
		prefix:     "",
		softTabs:   false,
		indentSize: 1,
		body:       body,
		header:     header,
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
func (this *Generator) Emit(sfmt string, args ...interface{}) {
	this.buffer.WriteString(fmt.Sprintf(sfmt, args...))
}

// Write a line, emitting an indent prefix beforehand.
func (this *Generator) Writeln(sfmt string, args ...interface{}) {
	this.EmitPrefix()
	this.Emit(sfmt, args...)
	this.Emit("\n")
}

// Emit an empty line.
func (this *Generator) Newline() {
	this.Emit("\n")
}

// Switch to emitting the header.
func (this *Generator) SwitchToHeader() {
	this.buffer = this.header
}

// Switch to emitting the body (default).
func (this *Generator) SwitchToBody() {
	this.buffer = this.body
}

func (this *Generator) ExportHeader(writer io.Writer) (int, error) {
	return writer.Write(this.header.Bytes())
}

func (this *Generator) ExportBody(writer io.Writer) (int, error) {
	return writer.Write(this.body.Bytes())
}
