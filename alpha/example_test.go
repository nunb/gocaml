package alpha

import (
	"fmt"
	"github.com/rhysd/gocaml/lexer"
	"github.com/rhysd/gocaml/parser"
	"github.com/rhysd/loc"
	"path/filepath"
)

func Example() {
	file := filepath.FromSlash("../testdata/from-mincaml/ack.ml")
	src, err := loc.NewSourceFromFile(file)
	if err != nil {
		// File not found
		panic(err)
	}

	lex := lexer.NewLexer(src)
	go lex.Lex()

	ast, err := parser.Parse(lex.Tokens)
	if err != nil {
		// When parse failed
		panic(err)
	}

	// Run alpha transform against the root of AST
	if err = Transform(ast.Root); err != nil {
		// When some some duplicates found
		panic(err)
	}

	// Now all symbols in the AST have unique names
	// e.g. abc -> abc$t1
	// And now all variable references (VarRef) point a symbol instance of the definition node.
	// By checking the pointer of symbol, we can know where the variable reference are defined
	// in source.
	fmt.Printf("%v\n", ast.Root)
}
