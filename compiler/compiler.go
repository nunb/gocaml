// Package compiler provides a compiler function for GoCaml codes.
package compiler

import (
	"fmt"
	"github.com/rhysd/gocaml/alpha"
	"github.com/rhysd/gocaml/ast"
	"github.com/rhysd/gocaml/closure"
	"github.com/rhysd/gocaml/codegen"
	"github.com/rhysd/gocaml/gcil"
	"github.com/rhysd/gocaml/lexer"
	"github.com/rhysd/gocaml/parser"
	"github.com/rhysd/gocaml/token"
	"github.com/rhysd/gocaml/typing"
	"github.com/rhysd/loc"
	"io/ioutil"
	"os"
	"path/filepath"
)

type OptLevel int

const (
	O0 OptLevel = iota
	O1
	O2
	O3
)

// Compiler instance to compile GoCaml code into other representations.
type Compiler struct {
	Optimization OptLevel
	LinkFlags    string
	TargetTriple string
	DebugInfo    bool
}

// PrintTokens returns the lexed tokens for a source code.
func (c *Compiler) Lex(src *loc.Source) chan token.Token {
	l := lexer.NewLexer(src)
	l.Error = func(msg string, pos loc.Pos) {
		fmt.Fprintf(os.Stderr, "%s at (line:%d, column:%d)\n", msg, pos.Line, pos.Column)
	}
	go l.Lex()
	return l.Tokens
}

// PrintTokens show list of tokens lexed.
func (c *Compiler) PrintTokens(src *loc.Source) {
	tokens := c.Lex(src)
	for {
		select {
		case t := <-tokens:
			fmt.Println(t.String())
			switch t.Kind {
			case token.EOF, token.ILLEGAL:
				return
			}
		}
	}
}

// Parse parses the source and returns the parsed AST.
func (c *Compiler) Parse(src *loc.Source) (*ast.AST, error) {
	tokens := c.Lex(src)
	ast, err := parser.Parse(tokens)

	if err != nil {
		return nil, err
	}

	ast.File = src // TODO

	return ast, nil
}

// PrintAST outputs AST structure to stdout.
func (c *Compiler) PrintAST(src *loc.Source) {
	a, err := c.Parse(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	ast.Println(a)
}

// SemanticAnalysis checks types and symbol duplicates.
// It returns the result of type analysis or an error.
func (c *Compiler) SemanticAnalysis(a *ast.AST) (*typing.Env, error) {
	if err := alpha.Transform(a.Root); err != nil {
		return nil, loc.Notef(err, "While semantic analysis (alpha transform) in %s\n", a.File.Path)
	}
	env, err := typing.TypeInferernce(a)
	if err != nil {
		return nil, loc.Notef(err, "While semantic analysis (type infererence) in %s", a.File.Path)
	}
	return env, nil
}

// EmitGCIL emits GCIL tree representation.
func (c *Compiler) EmitGCIL(src *loc.Source) (*gcil.Program, *typing.Env, error) {
	ast, err := c.Parse(src)
	if err != nil {
		return nil, nil, err
	}
	env, err := c.SemanticAnalysis(ast)
	if err != nil {
		return nil, nil, err
	}
	ir, err := gcil.FromAST(ast.Root, env)
	if err != nil {
		return nil, nil, err
	}
	gcil.ElimRefs(ir, env)
	prog := closure.Transform(ir)
	return prog, env, nil
}

func (c *Compiler) emitterFromSource(src *loc.Source) (*codegen.Emitter, error) {
	prog, env, err := c.EmitGCIL(src)
	if err != nil {
		return nil, err
	}

	level := codegen.OptimizeDefault
	switch c.Optimization {
	case O0:
		level = codegen.OptimizeNone
	case O1:
		level = codegen.OptimizeLess
	case O3:
		level = codegen.OptimizeAggressive
	}
	opts := codegen.EmitOptions{level, c.TargetTriple, c.LinkFlags, c.DebugInfo}

	return codegen.NewEmitter(prog, env, src, opts)
}

func (c *Compiler) EmitObjFile(src *loc.Source) error {
	emitter, err := c.emitterFromSource(src)
	if err != nil {
		return err
	}
	defer emitter.Dispose()
	emitter.RunOptimizationPasses()
	obj, err := emitter.EmitObject()
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s.o", src.BaseName())
	return ioutil.WriteFile(filename, obj, 0666)
}

func (c *Compiler) EmitLLVMIR(src *loc.Source) (string, error) {
	emitter, err := c.emitterFromSource(src)
	if err != nil {
		return "", err
	}
	defer emitter.Dispose()
	emitter.RunOptimizationPasses()

	return emitter.EmitLLVMIR(), nil
}

func (c *Compiler) EmitAsm(src *loc.Source) (string, error) {
	emitter, err := c.emitterFromSource(src)
	if err != nil {
		return "", err
	}
	defer emitter.Dispose()
	emitter.RunOptimizationPasses()

	return emitter.EmitAsm()
}

func (c *Compiler) Compile(source *loc.Source) error {
	emitter, err := c.emitterFromSource(source)
	if err != nil {
		return err
	}
	defer emitter.Dispose()
	emitter.RunOptimizationPasses()
	var executable string
	if source.Exists {
		executable = source.BaseName()
	} else {
		executable, err = filepath.Abs("a.out")
		if err != nil {
			return err
		}
	}
	return emitter.EmitExecutable(executable)
}
