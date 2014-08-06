package sema

import (
	. "github.com/edmodo/frugal/parser"
)

func checkUnused(context *CompileContext, tree *ParseTree) bool {
	for name, include := range tree.Includes {
		if _, ok := tree.UsedIncludes[name]; ok {
			continue
		}
		context.ReportError(
			include.Tok.Loc.Start,
			"include directive \"%s\" is unused",
			include.Tok.StringLiteral(),
		)
	}
	return !context.HasErrors()
}
