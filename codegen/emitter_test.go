package codegen

import (
	"github.com/rhysd/gocaml/alpha"
	"github.com/rhysd/gocaml/closure"
	"github.com/rhysd/gocaml/gcil"
	"github.com/rhysd/gocaml/lexer"
	"github.com/rhysd/gocaml/parser"
	"github.com/rhysd/gocaml/typing"
	"github.com/rhysd/loc"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testCreateEmitter(code string, optimize OptLevel, debug bool) (e *Emitter, err error) {
	s := loc.NewDummySource(code)
	l := lexer.NewLexer(s)
	go l.Lex()
	ast, err := parser.Parse(l.Tokens)
	if err != nil {
		return
	}
	if err = alpha.Transform(ast.Root); err != nil {
		return
	}
	env, err := typing.TypeInferernce(ast)
	if err != nil {
		return
	}
	ir, err := gcil.FromAST(ast.Root, env)
	if err != nil {
		return
	}
	gcil.ElimRefs(ir, env)
	prog := closure.Transform(ir)
	opts := EmitOptions{optimize, "", "", debug}
	e, err = NewEmitter(prog, env, s, opts)
	if err != nil {
		return
	}
	e.RunOptimizationPasses()
	return
}

func TestEmitLLVMIR(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeDefault, false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	ir := e.EmitLLVMIR()
	if !strings.Contains(ir, "ModuleID = '<dummy>'") {
		t.Fatalf("Module ID is not contained: %s", ir)
	}
	if !strings.Contains(ir, "target datalayout = ") {
		t.Fatalf("Data layout is not contained: %s", ir)
	}
}

func TestEmitAssembly(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeDefault, false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	asm, err := e.EmitAsm()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(asm, ".section") {
		t.Fatalf("Assembly was not emitted: %s", asm)
	}
}

func TestEmitObject(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeDefault, false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	obj, err := e.EmitObject()
	if err != nil {
		t.Fatal(err)
	}
	if len(obj) == 0 {
		t.Fatalf("Emitted object file is empty")
	}
}

func TestEmitExecutable(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeDefault, false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	outfile, err := filepath.Abs("__test_a.out")
	if err != nil {
		panic(err)
	}
	if err := e.EmitExecutable(outfile); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outfile)
	stats, err := os.Stat(outfile)
	if err != nil {
		t.Fatal("Cannot stat emitted executable", err)
	}
	if stats.IsDir() {
		t.Fatalf("File was not emitted actually")
	}
	if stats.Size() == 0 {
		t.Errorf("Emitted executable is empty")
	}
}

func TestEmitUnoptimizedLLVMIR(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeNone, false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	ir := e.EmitLLVMIR()
	if !strings.Contains(ir, `define private i64 @"f$t1"(i64 %"x$t2")`) {
		t.Fatal("Function 'f' was inlined with OptimizeNone config:", ir)
	}
}

func TestEmitLLVMIRWithDebugInfo(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeNone, true)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	ir := e.EmitLLVMIR()
	if !strings.Contains(ir, "!llvm.dbg.cu = ") {
		t.Fatalf("Debug information is not contained: %s", ir)
	}
}

func TestEmitOptimizedAggressive(t *testing.T) {
	e, err := testCreateEmitter("let rec f x = x + x in println_int (f 42)", OptimizeAggressive, false)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	ir := e.EmitLLVMIR()
	if strings.Contains(ir, `define private i64 @"f$t1"(i64 %"x$t2")`) {
		t.Fatalf("Function 'f' was not inlined with OptimizeAggressive config: %s", ir)
	}
}

func TestEmitIRContainingExternalSymbols(t *testing.T) {
	e, err := testCreateEmitter("x; y; f (x + y)", OptimizeDefault, true)
	if err != nil {
		t.Fatal(err)
	}
	defer e.Dispose()
	ir := e.EmitLLVMIR()
	expects := []string{
		"@x = external local_unnamed_addr global i64",
		"@y = external local_unnamed_addr global i64",
		"declare void @f(i64)",
	}
	for _, expect := range expects {
		if !strings.Contains(ir, expect) {
			t.Errorf("IR does not contain external symbol declaration '%s': %s", expect, ir)
		}
	}
}

func TestDisposeEmitter(t *testing.T) {
	e, err := testCreateEmitter("x; y; f (x + y); g (x < y)", OptimizeDefault, true)
	if err != nil {
		t.Fatal(err)
	}
	if e.Disposed {
		t.Fatal("Unexpectedly emitter was disposed")
	}
	e.Dispose()
	if !e.Disposed {
		t.Fatal("Emitter was not disposed by calling emitter.Dispose()")
	}
	// Do not crash when it's called twice
	e.Dispose()
}
