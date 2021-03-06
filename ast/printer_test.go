package ast

import (
	"bytes"
	"github.com/rhysd/gocaml/token"
	"github.com/rhysd/loc"
	"io"
	"os"
	"testing"
)

func TestPrintAST(t *testing.T) {
	s := loc.NewDummySource("")
	tok := &token.Token{
		Kind:  token.ILLEGAL,
		Start: loc.Pos{0, 0, 0, s},
		End:   loc.Pos{0, 0, 0, s},
		File:  s,
	}

	root := &Let{
		tok,
		NewSymbol("foo"),
		&Add{
			&Sub{
				&FSub{
					&Less{
						&Unit{tok, tok},
						&Not{
							tok,
							&Bool{tok, true},
						},
					},
					&Neg{
						tok,
						&Mod{
							&Int{tok, 42},
							&Int{tok, 42},
						},
					},
				},
				&Eq{
					&FNeg{
						tok,
						&Float{tok, 3.14},
					},
					&NotEq{
						&VarRef{tok, NewSymbol("variable")},
						&Int{tok, 42},
					},
				},
			},
			&FAdd{
				&FDiv{
					&Tuple{
						[]Expr{
							&Int{tok, 42},
							&Float{tok, 3.14},
							&ArraySize{
								tok,
								&VarRef{tok, NewSymbol("arr")},
							},
							&String{
								tok,
								"string literal",
							},
						},
					},
					&ArrayCreate{
						tok,
						&Int{tok, 42},
						&Typed{
							&Bool{tok, false},
							&CtorType{
								nil,
								tok,
								nil,
								"bool",
							},
						},
					},
				},
				&FMul{
					&Get{
						&ArrayCreate{
							tok,
							&Int{tok, 42},
							&Bool{tok, false},
						},
						&Int{tok, 1},
					},
					&Put{
						&ArrayCreate{
							tok,
							&Int{tok, 42},
							&Bool{tok, false},
						},
						&Int{tok, 1},
						&Bool{tok, true},
					},
				},
			},
		},
		&LetTuple{
			tok,
			[]*Symbol{
				NewSymbol("a"),
				NewSymbol("b"),
			},
			&Tuple{
				[]Expr{
					&Greater{
						&Int{tok, 1},
						&Int{tok, 2},
					},
					&GreaterEq{
						&Int{tok, 1},
						&Int{tok, 2},
					},
					&Mul{
						&Int{tok, 1},
						&Int{tok, 2},
					},
					&Div{
						&Int{tok, 1},
						&Int{tok, 2},
					},
					&And{
						&Bool{tok, true},
						&Bool{tok, false},
					},
					&Or{
						&Bool{tok, true},
						&Bool{tok, false},
					},
				},
			},
			&LetRec{
				tok,
				&FuncDef{
					NewSymbol("f"),
					[]Param{
						{
							NewSymbol("a"),
							&CtorType{
								nil,
								tok,
								nil,
								"unit",
							},
						},
					},
					&VarRef{tok, NewSymbol("a")},
					&CtorType{
						nil,
						tok,
						nil,
						"int",
					},
				},
				&If{
					tok,
					&LessEq{
						&Int{tok, 1},
						&Int{tok, 2},
					},
					&Apply{
						&VarRef{tok, NewSymbol("f")},
						[]Expr{
							&Int{tok, 42},
						},
					},
					&Match{
						tok,
						&Some{tok, &Int{tok, 1}},
						&None{tok},
						&None{tok},
						NewSymbol("foo"),
						tok.End,
					},
				},
			},
			&TupleType{
				[]Expr{
					&CtorType{
						nil,
						tok,
						[]Expr{
							&CtorType{
								nil,
								tok,
								nil,
								"unit",
							},
						},
						"foo",
					},
				},
			},
		},
		&FuncType{
			[]Expr{
				&CtorType{
					nil,
					tok,
					nil,
					"int",
				},
			},
			&CtorType{
				tok,
				tok,
				[]Expr{
					&CtorType{
						nil,
						tok,
						nil,
						"bool",
					},
					&CtorType{
						nil,
						tok,
						nil,
						"float",
					},
				},
				"foo",
			},
		},
	}

	ast := &AST{
		Root: root,
		File: s,
		TypeDecls: []*TypeDecl{
			{
				tok,
				"mytype",
				&CtorType{
					nil,
					tok,
					nil,
					"bool",
				},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Println(ast)

	ch := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		ch <- buf.String()
	}()
	w.Close()
	os.Stdout = old

	expected := `AST for <dummy>:
-   TypeDecl (mytype) (0:0-0:0)
-   -   CtorType (bool) (0:0-0:0)
-   Let (foo) (0:0-0:0)
-   -   FuncType (0:0-0:0)
-   -   -   CtorType (int) (0:0-0:0)
-   -   -   CtorType (foo (2)) (0:0-0:0)
-   -   -   -   CtorType (bool) (0:0-0:0)
-   -   -   -   CtorType (float) (0:0-0:0)
-   -   Add (0:0-0:0)
-   -   -   Sub (0:0-0:0)
-   -   -   -   FSub (0:0-0:0)
-   -   -   -   -   Less (0:0-0:0)
-   -   -   -   -   -   Unit (0:0-0:0)
-   -   -   -   -   -   Not (0:0-0:0)
-   -   -   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   -   Neg (0:0-0:0)
-   -   -   -   -   -   Mod (0:0-0:0)
-   -   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   Eq (0:0-0:0)
-   -   -   -   -   FNeg (0:0-0:0)
-   -   -   -   -   -   Float (0:0-0:0)
-   -   -   -   -   NotEq (0:0-0:0)
-   -   -   -   -   -   VarRef (variable) (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   FAdd (0:0-0:0)
-   -   -   -   FDiv (0:0-0:0)
-   -   -   -   -   Tuple (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   Float (0:0-0:0)
-   -   -   -   -   -   ArraySize (0:0-0:0)
-   -   -   -   -   -   -   VarRef (arr) (0:0-0:0)
-   -   -   -   -   -   String () (0:0-0:0)
-   -   -   -   -   ArrayCreate (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   Typed (0:0-0:0)
-   -   -   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   -   -   -   CtorType (bool) (0:0-0:0)
-   -   -   -   FMul (0:0-0:0)
-   -   -   -   -   Get (0:0-0:0)
-   -   -   -   -   -   ArrayCreate (0:0-0:0)
-   -   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Put (0:0-0:0)
-   -   -   -   -   -   ArrayCreate (0:0-0:0)
-   -   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   Bool (0:0-0:0)
-   -   LetTuple (a, b) (0:0-0:0)
-   -   -   TupleType (1) (0:0-0:0)
-   -   -   -   CtorType (foo (1)) (0:0-0:0)
-   -   -   -   -   CtorType (unit) (0:0-0:0)
-   -   -   Tuple (0:0-0:0)
-   -   -   -   Greater (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   GreaterEq (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   Mul (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   Div (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Int (0:0-0:0)
-   -   -   -   And (0:0-0:0)
-   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   Or (0:0-0:0)
-   -   -   -   -   Bool (0:0-0:0)
-   -   -   -   -   Bool (0:0-0:0)
-   -   -   LetRec (fun f a) (0:0-0:0)
-   -   -   -   CtorType (unit) (0:0-0:0)
-   -   -   -   CtorType (int) (0:0-0:0)
-   -   -   -   VarRef (a) (0:0-0:0)
-   -   -   -   If (0:0-0:0)
-   -   -   -   -   LessEq (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Apply (0:0-0:0)
-   -   -   -   -   -   VarRef (f) (0:0-0:0)
-   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   Match (foo) (0:0-0:0)
-   -   -   -   -   -   Some (0:0-0:0)
-   -   -   -   -   -   -   Int (0:0-0:0)
-   -   -   -   -   -   None (0:0-0:0)
-   -   -   -   -   -   None (0:0-0:0)
`
	actual := <-ch
	if expected != actual {
		t.Fatalf("Unexpected output from Println():\n%s", actual)
	}
}
