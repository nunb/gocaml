package typing

import (
	"github.com/rhysd/gocaml/alpha"
	"github.com/rhysd/gocaml/lexer"
	"github.com/rhysd/gocaml/parser"
	"github.com/rhysd/loc"
	"path/filepath"
	"strings"
	"testing"
)

func TestEdgeCases(t *testing.T) {
	testcases := []struct {
		what string
		code string
	}{
		{
			what: "param and function have the same name",
			code: "let rec f f = f + 1 in print_int (f 10)",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.what, func(t *testing.T) {
			s := loc.NewDummySource(tc.code)
			l := lexer.NewLexer(s)
			go l.Lex()
			ast, err := parser.Parse(l.Tokens)
			if err != nil {
				panic(err)
			}
			if err = alpha.Transform(ast.Root); err != nil {
				panic(err)
			}
			i := NewInferer()
			i.conv, err = newNodeTypeConv(ast.TypeDecls)
			if err != nil {
				t.Fatal(err)
			}
			_, err = i.infer(ast.Root)
			if err != nil {
				t.Fatalf("Type check raised an error for code '%s': %s", tc.code, err.Error())
			}
		})
	}
}

func TestInvalidExpressions(t *testing.T) {
	testcases := []struct {
		what     string
		code     string
		expected string
	}{
		{
			what:     "+. with int",
			code:     "1 +. 2",
			expected: "Type mismatch between 'float' and 'int'",
		},
		{
			what:     "+ with float",
			code:     "1.0 + 2.0",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "/ with float",
			code:     "1.0 / 2.0",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "% with float",
			code:     "1.0 % 2.0",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "'not' with non-bool value",
			code:     "not 42",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "invalid equal compare",
			code:     "41 = true",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "invalid = compare",
			code:     "41 = 3.14",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "invalid <> compare",
			code:     "41 <> 3.14",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "invalid < compare",
			code:     "41 < true",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "invalid <= compare",
			code:     "41 <= true",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "invalid > compare",
			code:     "41 > true",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "invalid >= compare",
			code:     "41 >= true",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "&& must have boolean operands",
			code:     "42 && true",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "|| must have boolean operands",
			code:     "false || 42",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "&& is evaluated as bool",
			code:     "(true && false) + 3",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "/. with int",
			code:     "1 /. 2",
			expected: "Type mismatch between 'float' and 'int'",
		},
		{
			what:     "*. with int",
			code:     "1 *. 2",
			expected: "Type mismatch between 'float' and 'int'",
		},
		{
			what:     "unary - without number",
			code:     "-true",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "unary -. with non-float",
			code:     "-.42",
			expected: "operand of unary operator '-.' must be 'float'",
		},
		{
			what:     "not a bool condition in if",
			code:     "if 42 then true else false",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "mismatch type between else and then",
			code:     "if true then 42 else 4.2",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "mismatch type of variable",
			code:     "let x = true in x + 42",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "mismatch parameter type",
			code:     "let rec f a b = a < b in (f 1 1) = (f 1.0 1.0)",
			expected: "On unifying 1st parameter of function 'int -> int -> bool' and 'float -> float -> bool'",
		},
		{
			what:     "does not meet parameter type requirements",
			code:     "let rec f a b = a + b in f 1 1.0",
			expected: "On unifying 2nd parameter of function 'int -> int -> int' and 'int -> float -> int'",
		},
		{
			what:     "wrong number of arguments",
			code:     "let rec f a b = a + b in f 1",
			expected: "Number of parameters of function does not match: 2 vs 1 (between 'int -> int -> int' and 'int -> int')",
		},
		{
			what:     "type mismatch in return type",
			code:     "let rec f a b = a + b in 1.0 +. f 1 2",
			expected: "Type mismatch between 'float' and 'int'",
		},
		{
			what:     "wrong number of tuple assignment",
			code:     "let (x, y) = (1, 2, 3) in ()",
			expected: "Number of elements of tuple does not match",
		},
		{
			what:     "type mismatch for tuple elements",
			code:     "let (x, y) = (1, 2.0) in x + y",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "index is not a number",
			code:     "let a = Array.make 3 1.0 in a.(true)",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "wrong array length type",
			code:     "let a = Array.make true 1.0 in ()",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "element type mismatch in array",
			code:     "let a = Array.make 3 1.0 in 1 + a.(0)",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "index access to wrong value",
			code:     "true.(1)",
			expected: "array' and 'bool'",
		},
		{
			what:     "set wrong type value to array",
			code:     "let a = Array.make 3 1.0 in a.(0) <- true",
			expected: "Type mismatch between 'bool' and 'float'",
		},
		{
			what:     "wrong index type in index access",
			code:     "let a = Array.make 3 1.0 in a.(true) <- 2.0",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "index assign to wrong value",
			code:     "false.(1) <- 10",
			expected: "Type mismatch between 'int array' and 'bool'",
		},
		{
			what:     "index assign is evaluated as unit",
			code:     "let a = Array.make 3 1.0 in 1.0 = a.(0) <- 2.0",
			expected: "Type mismatch between 'float' and '()'",
		},
		{
			what:     "Array.length with invalid argument",
			code:     "Array.length true",
			expected: "array' and 'bool'",
		},
		{
			what:     "Array.length returns int type value",
			code:     "(Array.length (Array.make 3 true)) = 3.0",
			expected: "'int' and 'float'",
		},
		{
			what:     "occur check",
			code:     "let rec f x = f in f 4",
			expected: "Cyclic dependency found while unification with",
		},
		{
			what:     "pre-registered external functions (param type)",
			code:     "println_bool 42",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "pre-registered external functions (return type)",
			code:     `println_bool (str_length "foo")`,
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "'argv' special global variable",
			code:     "argv + 12",
			expected: "Type mismatch between 'int' and 'string array'",
		},
		{
			what:     "Option type",
			code:     "let a = Some 42 in let b = Some true in a = b",
			expected: "Type mismatch between 'int' and 'bool'",
		},
		{
			what:     "matching target in match expression",
			code:     "match 42 with Some i -> 0 | None -> 0",
			expected: "matching target in 'match' expression must be '?",
		},
		{
			what:     "matched symbol type and matching expression",
			code:     "match Some 42 with Some i -> not i | None -> false",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "match expression arms",
			code:     "match Some 42 with Some i -> 3.14 | None -> true",
			expected: "mismatch of types between 'Some' arm and 'None' arm in 'match' expression",
		},
		{
			what:     "None type comparison",
			code:     "let o = None in o = 42",
			expected: "option' and 'int'",
		},
		{
			what:     "Invalid type specified",
			code:     "let foo: fooooooo = 42 in foo",
			expected: "Unknown type constructor 'fooooooo'",
		},
		{
			what:     "Type mismatch at type annotation",
			code:     "let foo: bool = 42 in foo",
			expected: "Type mismatch between 'bool' and 'int'",
		},
		{
			what:     "Type mismatch at type annotation (let tuple)",
			code:     "let (x, y): int * bool = 42, 3.14 in x",
			expected: "Type mismatch between 'bool' and 'float'",
		},
		{
			what:     "'let tuple' must annotated as tuple",
			code:     "let (x, y): bool option = 42, 3.14 in x",
			expected: "must be tuple, but found 'bool option'",
		},
		{
			what:     "Number of tuple elements mismatch at 'let tuple'",
			code:     "let (x, y): int * bool * float = 42, false in x",
			expected: "3 vs 2",
		},
		{
			what:     "Type mismatch at (e: ty) expression",
			code:     "let i = 42 in (i: bool)",
			expected: "mismatch between inferred type and specified type",
		},
		{
			what:     "Invalid type at (e: ty) expression",
			code:     "(i: foooo)",
			expected: "Unknown type constructor 'foooo'",
		},
		{
			what:     "Type mismatch at param type",
			code:     "let rec f (x:float) = -x in f",
			expected: "Type mismatch between 'int' and 'float'",
		},
		{
			what:     "Type mismatch at return type",
			code:     "let rec f (x:int): float = x in f",
			expected: "return type of function",
		},
		{
			what:     "Invalid parameter type",
			code:     "let rec f (x:(int, int) array) = x in f",
			expected: "1st parameter of function",
		},
		{
			what:     "Invalid return type",
			code:     "let rec f x: foo = x in f",
			expected: "return type of function",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.what, func(t *testing.T) {
			s := loc.NewDummySource(testcase.code)
			l := lexer.NewLexer(s)
			go l.Lex()
			ast, err := parser.Parse(l.Tokens)
			if err != nil {
				panic(err)
			}
			if err = alpha.Transform(ast.Root); err != nil {
				panic(err)
			}
			i := NewInferer()
			i.conv, err = newNodeTypeConv(ast.TypeDecls)
			if err != nil {
				t.Fatal(err)
			}
			_, err = i.infer(ast.Root)
			if err == nil {
				t.Fatalf("Type check did not raise an error for code '%s'", testcase.code)
			}
			if !strings.Contains(err.Error(), testcase.expected) {
				t.Fatalf("Expected error message '%s' to contain '%s'", err.Error(), testcase.expected)
			}
		})
	}
}

func TestRegisterNoneTypes(t *testing.T) {
	s := loc.NewDummySource("let rec f x = () in f (Some 42); f None; let a = None in f a")
	l := lexer.NewLexer(s)
	go l.Lex()
	ast, err := parser.Parse(l.Tokens)
	if err != nil {
		panic(err)
	}
	if err = alpha.Transform(ast.Root); err != nil {
		panic(err)
	}
	i := NewInferer()
	i.conv, err = newNodeTypeConv(ast.TypeDecls)
	if err != nil {
		t.Fatal(err)
	}
	_, err = i.infer(ast.Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(i.env.NoneTypes) != 2 {
		t.Errorf("2 None node should be detected but actually %d", len(i.env.NoneTypes))
	}
}

func TestInferSuccess(t *testing.T) {
	files, err := filepath.Glob("testdata/*.ml")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			s, err := loc.NewSourceFromFile(file)
			if err != nil {
				panic(err)
			}
			l := lexer.NewLexer(s)
			go l.Lex()
			ast, err := parser.Parse(l.Tokens)
			if err != nil {
				t.Fatal(err)
			}
			if err = alpha.Transform(ast.Root); err != nil {
				t.Fatal(err)
			}
			i := NewInferer()
			i.conv, err = newNodeTypeConv(ast.TypeDecls)
			if err != nil {
				t.Fatal(err)
			}
			_, err = i.infer(ast.Root)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
