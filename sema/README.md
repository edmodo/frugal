frugal/sema
=============

Semantic analysis for thrift IDL files. This package has tools to make sure that all name references are bound, that types are checked and are non-circular, that names are not duplicated, et cetera.

Example:

```
import (
  "github.com/edmodo/frugal/parser"
  "github.com/edmodo/frugal/sema"
)

func ParseAndAnalyze(file string)  {
  context := parser.NewCompileContext()
  tree := context.ParseRecursive(file)
  if tree == nil {
    context.PrintErrors()
    return
  }
  if !sema.Analyze(context, tree) {
    context.PrintErrors()
    return
  }
}
```
