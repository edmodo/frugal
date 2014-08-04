package parser

import (
	"fmt"
)

type CompileError struct {
	File    string
	Pos     Position
	Message string
}

type CompileContext struct {
	// Current file being operated on, if any.
	CurFile string

	// List of errors encountered so far.
	Errors  []*CompileError
}

func NewCompileContext() *CompileContext {
	return &CompileContext{}
}

func (this *CompileContext) Enter(file string) {
	if this.CurFile != "" {
		panic("Cannot nested files")
	}
	this.CurFile = file
}

func (this *CompileContext) Leave() {
	this.CurFile = ""
}

func (this *CompileContext) HasErrors() bool {
	return len(this.Errors) > 0
}

func (this *CompileContext) ReportError(pos Position, str string, args ...interface{}) {
	this.Errors = append(this.Errors, &CompileError{
		File:    this.CurFile,
		Pos:     pos,
		Message: fmt.Sprintf(str, args...),
	})
}

func (this *CompileContext) PrintErrors() {
	for _, err := range this.Errors {
		fmt.Printf("%s (line %d, col %d): %s\n", err.File, err.Pos.Line, err.Pos.Col, err.Message)
	}
}
