// Package typing provides type inference logic for GoCaml.
package typing

import (
	"fmt"
	"github.com/rhysd/gocaml/ast"
)

// Result of type analysis.
type Env struct {
	// Types for declarations. This is referred by type variables to resolve
	// type variables' actual types
	//
	// XXX:
	// Currently nested identifiers don't work. Example:
	//   let
	//     x = 42
	//   in
	//     let x = true in print_bool (x);
	//     print_int (x)
	// We need alpha transform before type inference in order to ensure
	// all symbol names are unique.
	Table map[string]Type
	// External variable names which are referred but not defined.
	// External variables are exposed as external symbols in other object files.
	Externals map[string]Type
	// Type of `None` will be inferred. To know what type the `None` values is typed,
	// we need to memorize them in type inference.
	NoneTypes map[*ast.None]*Option
}

// NewEnv creates empty Env instance.
func NewEnv() *Env {
	return &Env{
		map[string]Type{},
		builtinPopulatedTable(),
		map[*ast.None]*Option{},
	}
}

func (env *Env) Dump() {
	fmt.Println("Variables:")
	for s, t := range env.Table {
		fmt.Printf("  %s: %s\n", s, t.String())
	}
	fmt.Println()
	env.DumpExternals()
}

func (env *Env) DumpExternals() {
	fmt.Println("External Variables:")
	for s, t := range env.Externals {
		fmt.Printf("  %s: %s\n", s, t.String())
	}
}
