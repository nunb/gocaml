package ast

import (
	"fmt"
	"github.com/rhysd/gocaml/token"
	"github.com/rhysd/loc"
	"path/filepath"
)

// Visitor which counts number of nodes in AST
type numAllNodes struct {
	total int
}

// Visit method to meets ast.Visitor interface
func (v *numAllNodes) Visit(e Expr) Visitor {
	v.total++
	return v
}

func Example() {
	file := filepath.FromSlash("../testdata/from-mincaml/ack.ml")
	src, err := loc.NewSourceFromFile(file)
	if err != nil {
		// File not found
		panic(err)
	}

	// AST which usually comes from parser.Parse() function.
	rootOfAST := &Let{
		LetToken: &token.Token{},
		Symbol:   NewSymbol("test"),
		Bound: &Int{
			Token: &token.Token{},
			Value: 42,
		},
		Body: &Add{
			Left: &VarRef{
				Token:  &token.Token{},
				Symbol: NewSymbol("test"),
			},
			Right: &Float{
				Token: &token.Token{},
				Value: 3.14,
			},
		},
	}

	ast := &AST{
		File: src,
		Root: rootOfAST,
	}

	// Apply visitor to root node of AST
	v := &numAllNodes{0}
	Visit(v, ast.Root)
	fmt.Println(v.total)
	// Output: 5

	// Print AST
	Println(ast)
}
