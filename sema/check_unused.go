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
