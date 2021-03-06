package gcil

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/rhysd/gocaml/alpha"
	"github.com/rhysd/gocaml/lexer"
	"github.com/rhysd/gocaml/parser"
	"github.com/rhysd/gocaml/typing"
	"github.com/rhysd/loc"
	"strings"
	"testing"
)

func TestEmitInsn(t *testing.T) {
	cases := []struct {
		what     string
		code     string
		expected []string
	}{
		{
			"int",
			"42",
			[]string{"int 42 ; type=int"},
		},
		{
			"unit",
			"()",
			[]string{"unit ; type=()"},
		},
		{
			"float",
			"3.14",
			[]string{"float 3.140000 ; type=float"},
		},
		{
			"boolean",
			"false",
			[]string{"bool false ; type=bool"},
		},
		{
			"string",
			`"this is\ttest\n"`,
			[]string{`string "this is\ttest\n" ; type=string`},
		},
		{
			"unary relational op",
			"not true",
			[]string{
				"bool true ; type=bool",
				"unary not $k1 ; type=bool",
			},
		},
		{
			"unary arithmetic op",
			"-42; -.1.0",
			[]string{
				"int 42 ; type=int",
				"unary - $k1 ; type=int",
				"float 1.000000 ; type=float",
				"unary -. $k3 ; type=float",
			},
		},
		{
			"binary int op",
			"1 + 2; 1 * 2; 1 / 2",
			[]string{
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary + $k1 $k2 ; type=int",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary * $k4 $k5 ; type=int",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary / $k7 $k8 ; type=int",
			},
		},
		{
			"binary float op",
			"3.14 *. 2.0; 3.14 +. 2.0; 3.14 -. 2.0; 3.14 /. 2.0",
			[]string{
				"float 3.140000 ; type=float",
				"float 2.000000 ; type=float",
				"binary *. $k1 $k2 ; type=float",
				"float 3.140000 ; type=float",
				"float 2.000000 ; type=float",
				"binary +. $k4 $k5 ; type=float",
				"float 3.140000 ; type=float",
				"float 2.000000 ; type=float",
				"binary -. $k7 $k8 ; type=float",
				"float 3.140000 ; type=float",
				"float 2.000000 ; type=float",
				"binary /. $k10 $k11 ; type=float",
			},
		},
		{
			"binary relational op",
			"1 < 2; 1 = 2; 1 <= 2; 1 > 2; 1 >= 2; 1 <> 2",
			[]string{
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary < $k1 $k2 ; type=bool",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary = $k4 $k5 ; type=bool",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary <= $k7 $k8 ; type=bool",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary > $k10 $k11 ; type=bool",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary >= $k13 $k14 ; type=bool",
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary <> $k16 $k17 ; type=bool",
			},
		},
		{
			"binary logical op",
			"true && false; true || false",
			[]string{
				"bool true ; type=bool",
				"bool false ; type=bool",
				"binary && $k1 $k2 ; type=bool",
				"bool true ; type=bool",
				"bool false ; type=bool",
				"binary || $k4 $k5 ; type=bool",
			},
		},
		{
			"if expression",
			"if 1 < 2 then 3 else 4",
			[]string{
				"int 1 ; type=int",
				"int 2 ; type=int",
				"binary < $k1 $k2 ; type=bool",
				"if $k3 ; type=int",
				"BEGIN: then",
				"int 3 ; type=int",
				"END: then",
				"BEGIN: else",
				"int 4 ; type=int",
				"END: else",
			},
		},
		{
			"let expression and variable reference",
			"let a = 1 in let b = a in b",
			[]string{
				"int 1 ; type=int",
				"ref a$t1 ; type=int",
				"ref b$t2 ; type=int",
			},
		},
		{
			"function and its application",
			"let rec f a = a + 1 in f 3",
			[]string{
				"fun a$t2 ; type=int -> int",
				"BEGIN: body (f$t1)",
				"ref a$t2 ; type=int",
				"int 1 ; type=int",
				"binary + $k1 $k2 ; type=int",
				"END: body (f$t1)",
				"ref f$t1 ; type=int -> int",
				"int 3 ; type=int",
				"app $k4 $k5 ; type=int",
			},
		},
		{
			"tuple literal",
			"(1, 2, 3)",
			[]string{
				"int 1 ; type=int",
				"int 2 ; type=int",
				"int 3 ; type=int",
				"tuple $k1,$k2,$k3 ; type=int * int * int",
			},
		},
		{
			"let tuple substitution",
			"let (a, b) = (1, 2) in a + b",
			[]string{
				"int 1 ; type=int",
				"int 2 ; type=int",
				"tuple $k1,$k2 ; type=int * int",
				"tplload 0 $k3 ; type=int",
				"tplload 1 $k3 ; type=int",
				"ref a$t1 ; type=int",
				"ref b$t2 ; type=int",
				"binary + $k4 $k5 ; type=int",
			},
		},
		{
			"array creation",
			"Array.make 3 true",
			[]string{
				"int 3 ; type=int",
				"bool true ; type=bool",
				"array $k1 $k2 ; type=bool array",
			},
		},
		{
			"array size",
			"Array.length (Array.make 3 true)",
			[]string{
				"int 3 ; type=int",
				"bool true ; type=bool",
				"array $k1 $k2 ; type=bool array",
				"arrlen $k3 ; type=int",
			},
		},
		{
			"access to array",
			"let a = Array.make 3 true in a.(1)",
			[]string{
				"int 3 ; type=int",
				"bool true ; type=bool",
				"array $k1 $k2 ; type=bool array",
				"ref a$t1 ; type=bool array",
				"int 1 ; type=int",
				"arrload $k5 $k4 ; type=bool",
			},
		},
		{
			"modify element of array",
			"let a = Array.make 3 true in a.(1) <- false",
			[]string{
				"int 3 ; type=int",
				"bool true ; type=bool",
				"array $k1 $k2 ; type=bool array",
				"ref a$t1 ; type=bool array",
				"int 1 ; type=int",
				"bool false ; type=bool",
				"arrstore $k5 $k4 $k6 ; type=bool",
			},
		},
		{
			"external symbol references",
			"x + 0",
			[]string{
				"xref x ; type=int",
			},
		},
		{
			"external symbol references 2",
			"x < 3",
			[]string{
				"xref x ; type=int",
			},
		},
		{
			"sequential expression",
			"1; true; 1.0",
			[]string{
				"int 1 ; type=int",
				"bool true ; type=bool",
				"float 1.000000 ; type=float",
			},
		},
		{
			"nested blocks",
			"if true then if false then 1 else 2 else 3",
			[]string{
				"bool true ; type=bool",
				"if $k1 ; type=int",
				"BEGIN: then",
				"bool false ; type=bool",
				"if $k2 ; type=int",
				"BEGIN: then",
				"int 1 ; type=int",
				"END: then",
				"BEGIN: else",
				"int 2 ; type=int",
				"END: else",
				"END: then",
				"BEGIN: else",
				"int 3 ; type=int",
				"END: else",
			},
		},
		{
			"option value",
			"if true then None else Some 42",
			[]string{
				"bool true ; type=bool",
				"if $k1 ; type=int option",
				"BEGIN: then",
				"none ; type=int option",
				"END: then",
				"BEGIN: else",
				"int 42 ; type=int",
				"some $k3 ; type=int option",
				"END: else",
			},
		},
		{
			"match with some value",
			"match Some 42 with Some i -> i + 3 | None -> 42",
			[]string{
				"int 42 ; type=int",
				"some $k1 ; type=int option",
				"issome $k2 ; type=bool",
				"if $k3 ; type=int",
				"BEGIN: then",
				"i$t1 = derefsome $k2 ; type=int",
				"ref i$t1 ; type=int",
				"int 3 ; type=int",
				"binary + $k4 $k5 ; type=int",
				"END: then",
				"BEGIN: else",
				"int 42 ; type=int",
				"END: else",
			},
		},
		{
			"match with none value",
			"match None with Some i -> i | None -> false",
			[]string{
				"none ; type=bool option",
				"issome $k1 ; type=bool",
				"if $k2 ; type=bool",
				"BEGIN: then",
				"i$t1 = derefsome $k1 ; type=bool",
				"ref i$t1 ; type=bool",
				"END: then",
				"BEGIN: else",
				"bool false ; type=bool",
				"END: else",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.what, func(t *testing.T) {
			s := loc.NewDummySource(fmt.Sprintf("%s; ()", tc.code))
			l := lexer.NewLexer(s)
			go l.Lex()
			ast, err := parser.Parse(l.Tokens)
			if err != nil {
				t.Fatal(err)
			}
			if err = alpha.Transform(ast.Root); err != nil {
				t.Fatal(err)
			}
			env, err := typing.TypeInferernce(ast)
			if err != nil {
				t.Fatal(err)
			}
			ir, err := FromAST(ast.Root, env)
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			ir.Println(&buf, env)
			r := bufio.NewReader(&buf)
			line, _, err := r.ReadLine()
			if err != nil {
				t.Fatal(err)
			}
			if string(line) != "BEGIN: program" {
				t.Fatalf("First line must begin with 'BEGIN: program' because it's root block")
			}
			for i, expected := range tc.expected {
				line, _, err = r.ReadLine()
				if err != nil {
					t.Fatalf("At line %d of output of ir for code '%s'", i, tc.code)
				}
				actual := string(line)
				if !strings.HasSuffix(actual, expected) {
					t.Errorf("Expected to end with '%s' for line %d of output of code '%s'. But actually output was '%s'", expected, i, tc.code, actual)
				}
			}
		})
	}
}

func TestSemanticError(t *testing.T) {
	cases := []struct {
		what     string
		code     string
		expected string
	}{
		{
			what:     "unit is invalid for operator '<'",
			code:     "() < ()",
			expected: "'()' can't be compared with operator '<'",
		},
		{
			what:     "tuple is invalid for operator '<'",
			code:     "let t = (1, 2) in t < t",
			expected: "'int * int' can't be compared with operator '<'",
		},
		{
			what:     "option is invalid for operator '<'",
			code:     "let a = Some 3 in a < None",
			expected: "'int option' can't be compared with operator '<'",
		},
		{
			what:     "array is invalid for operator '='",
			code:     "let a = Array.make  3 3 in a = a",
			expected: "'int array' can't be compared with operator '='",
		},
	}

	for _, tc := range cases {
		t.Run(tc.what, func(t *testing.T) {
			s := loc.NewDummySource(fmt.Sprintf("%s; ()", tc.code))
			l := lexer.NewLexer(s)
			go l.Lex()
			ast, err := parser.Parse(l.Tokens)
			if err != nil {
				t.Fatal(err)
			}
			if err = alpha.Transform(ast.Root); err != nil {
				t.Fatal(err)
			}
			env, err := typing.TypeInferernce(ast)
			if err != nil {
				t.Fatal(err)
			}
			_, err = FromAST(ast.Root, env)
			if err == nil {
				t.Fatalf("Expected code '%s' to cause an error '%s' but actually there is no error", tc.code, tc.expected)
			}
			if !strings.Contains(err.Error(), tc.expected) {
				t.Fatalf("Error message '%s' does not contain '%s'", err.Error(), tc.expected)
			}
		})
	}
}
