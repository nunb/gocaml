// Package lexer provides an instance for lexing GoCaml codes.
package lexer

import (
	"bytes"
	"fmt"
	"github.com/rhysd/gocaml/token"
	"io"
	"unicode"
	"unicode/utf8"
)

type stateFn func(*Lexer) stateFn

const eof = -1

// Lexer instance which contains lexing states.
type Lexer struct {
	state   stateFn
	start   token.Position
	current token.Position
	src     *token.Source
	input   *bytes.Reader
	Tokens  chan token.Token
	top     rune
	eof     bool
	// Function called when error occurs.
	// By default it outputs an error to stderr.
	Error func(msg string, pos token.Position)
}

// NewLexer creates new Lexer instance.
func NewLexer(src *token.Source) *Lexer {
	start := token.Position{
		Offset: 0,
		Line:   1,
		Column: 1,
	}
	return &Lexer{
		state:   lex,
		start:   start,
		current: start,
		input:   bytes.NewReader(src.Code),
		src:     src,
		Tokens:  make(chan token.Token),
		Error:   nil,
	}
}

// Lex starts lexing. Lexed tokens will be queued into channel in lexer.
func (l *Lexer) Lex() {
	// Set top to peek current rune
	l.forward()
	for l.state != nil {
		l.state = l.state(l)
	}
}

func (l *Lexer) emit(kind token.Kind) {
	l.Tokens <- token.Token{
		kind,
		l.start,
		l.current,
		l.src,
	}
	l.start = l.current
}

func (l *Lexer) emitIdent(ident string) {
	if len(ident) == 1 {
		// Shortcut because no keyword is one character. It must be identifier
		l.emit(token.IDENT)
		return
	}

	switch ident {
	case "true", "false":
		l.emit(token.BOOL)
	case "if":
		l.emit(token.IF)
	case "then":
		l.emit(token.THEN)
	case "else":
		l.emit(token.ELSE)
	case "let":
		l.emit(token.LET)
	case "in":
		l.emit(token.IN)
	case "rec":
		l.emit(token.REC)
	case "not":
		l.emit(token.NOT)
	default:
		l.emit(token.IDENT)
	}
}

func (l *Lexer) emitIllegal() {
	t := token.Token{
		token.ILLEGAL,
		l.start,
		l.current,
		l.src,
	}
	l.Tokens <- t
	l.start = l.current
}

func (l *Lexer) expected(s string, actual rune) {
	l.errmsg(fmt.Sprintf("Expected %s but got '%c'(%d)", s, actual, actual))
	l.emitIllegal()
}

func (l *Lexer) unclosedComment(expected string) {
	l.errmsg(fmt.Sprintf("Expected '%s' for closing comment but got EOF", expected))
	l.emitIllegal()
}

func (l *Lexer) forward() {
	r, _, err := l.input.ReadRune()
	if err == io.EOF {
		l.top = 0
		l.eof = true
		return
	}

	if err != nil {
		panic(err)
	}

	if !utf8.ValidRune(r) {
		panic(fmt.Errorf("Invalid UTF-8 character at line:%d,col:%d: '%c' (%d)", l.current.Line, l.current.Column, r, r))
	}

	l.top = r
	l.eof = false
}

func (l *Lexer) eat() {
	size := utf8.RuneLen(l.top)
	l.current.Offset += size

	// TODO: Consider \n\r
	if l.top == '\n' {
		l.current.Line++
		l.current.Column = 1
	} else {
		l.current.Column += size
	}

	l.forward()
}

func (l *Lexer) consume() {
	if l.eof {
		return
	}
	l.eat()
	l.start = l.current
}

func (l *Lexer) errmsg(msg string) {
	if l.Error == nil {
		return
	}
	l.Error(msg, l.current)
}

func (l *Lexer) eatIndent() bool {
	if !isLetter(l.top) {
		l.expected("letter for head character of identifer", l.top)
		return false
	}
	l.eat()

	for isLetter(l.top) || unicode.IsDigit(l.top) {
		l.eat()
	}
	return true
}

func lexComment(l *Lexer) stateFn {
	for {
		if l.eof {
			l.unclosedComment("*")
			return nil
		}
		if l.top == '*' {
			l.eat()
			if l.eof {
				l.unclosedComment(")")
				return nil
			}
			if l.top == ')' {
				l.eat()
				l.emit(token.COMMENT)
				return lex
			}
		}
		l.eat()
	}
}

func lexLeftParen(l *Lexer) stateFn {
	l.eat()
	if l.top == '*' {
		l.eat()
		return lexComment
	}
	l.emit(token.LPAREN)
	return lex
}

func lexAdditiveOp(l *Lexer) stateFn {
	dot, op := token.PLUS_DOT, token.PLUS
	if l.top == '-' {
		dot, op = token.MINUS_DOT, token.MINUS
	}
	l.eat()

	if l.top == '.' {
		l.eat()
		l.emit(dot)
	} else {
		l.emit(op)
	}
	return lex
}

func lexMultOp(l *Lexer) stateFn {
	op, dot := token.STAR, token.STAR_DOT
	if l.top == '/' {
		op, dot = token.SLASH, token.SLASH_DOT
	}
	l.eat()

	if l.top == '.' {
		l.eat()
		l.emit(dot)
	} else {
		l.emit(op)
	}

	return lex
}

func lexLess(l *Lexer) stateFn {
	l.eat()
	switch l.top {
	case '>':
		l.eat()
		l.emit(token.LESS_GREATER)
	case '=':
		l.eat()
		l.emit(token.LESS_EQUAL)
	case '-':
		l.eat()
		l.emit(token.LESS_MINUS)
	default:
		l.emit(token.LESS)
	}
	return lex
}

func lexGreater(l *Lexer) stateFn {
	l.eat()
	switch l.top {
	case '=':
		l.eat()
		l.emit(token.GREATER_EQUAL)
	default:
		l.emit(token.GREATER)
	}
	return lex
}

// e.g. 123.45e10
func lexNumber(l *Lexer) stateFn {
	tok := token.INT

	// Eat first digit. It's known as digit in lex()
	l.eat()
	for unicode.IsDigit(l.top) {
		l.eat()
	}

	// Note: Allow 1. as 1.0
	if l.top == '.' {
		tok = token.FLOAT
		l.eat()
		for unicode.IsDigit(l.top) {
			l.eat()
		}
	}

	if l.top == 'e' || l.top == 'E' {
		tok = token.FLOAT
		l.eat()
		if l.top == '+' || l.top == '-' {
			l.eat()
		}
		if !unicode.IsDigit(l.top) {
			l.expected("number for exponential part of float literal", l.top)
			return nil
		}
		for unicode.IsDigit(l.top) {
			l.eat()
		}
	}

	l.emit(tok)
	return lex
}

func isLetter(r rune) bool {
	return 'a' <= r && r <= 'z' ||
		'A' <= r && r <= 'Z' ||
		r == '_' ||
		r >= utf8.RuneSelf && unicode.IsLetter(r)
}

func lexArrayCreate(l *Lexer) stateFn {
	if l.top != '.' {
		l.expected("'.' for 'Array.create'", l.top)
		return nil
	}
	l.eat()

	if !l.eatIndent() {
		return nil
	}

	ident := string(l.src.Code[l.start.Offset:l.current.Offset])

	switch ident {
	// Note:
	// Ate 'Array' and '.' but no token was emitted. So 'Array.' remains as
	// current token string.
	case "Array.create", "Array.make":
		l.emit(token.ARRAY_CREATE)
		return lex
	default:
		l.errmsg(fmt.Sprintf("Expected 'create' or 'make' for Array.create but got '%s'", ident))
		return nil
	}
}

func lexIdent(l *Lexer) stateFn {
	if !l.eatIndent() {
		return nil
	}
	i := string(l.src.Code[l.start.Offset:l.current.Offset])
	if i == "Array" {
		return lexArrayCreate
	}
	l.emitIdent(i)
	return lex
}

func lex(l *Lexer) stateFn {
	for {
		if l.eof {
			l.emit(token.EOF)
			return nil
		}
		switch l.top {
		case '(':
			return lexLeftParen
		case ')':
			l.eat()
			l.emit(token.RPAREN)
		case '+':
			return lexAdditiveOp
		case '-':
			return lexAdditiveOp
		case '*':
			return lexMultOp
		case '/':
			return lexMultOp
		case '=':
			l.eat()
			l.emit(token.EQUAL)
		case '<':
			return lexLess
		case '>':
			return lexGreater
		case ',':
			l.eat()
			l.emit(token.COMMA)
		case '.':
			l.eat()
			l.emit(token.DOT)
		case ';':
			l.eat()
			l.emit(token.SEMICOLON)
		default:
			switch {
			case unicode.IsSpace(l.top):
				l.consume()
			case unicode.IsDigit(l.top):
				return lexNumber
			default:
				return lexIdent
			}
		}
	}
}
