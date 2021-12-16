package main

import (
	"github.com/ZadenRB/go-lexer"
	"regexp"
	"strconv"
)

/*
	Comments with "x.y.z - Rule n" format refer to the POSIX Shell Command Language Spec
	https://pubs.opengroup.org/onlinepubs/9699919799/utilities/V3_chap02.html
	where x.y.z is the section, and n is the number of a rule within that section.
*/

var commentPattern = regexp.MustCompile("[^\n]")
var stringLiteralPattern = regexp.MustCompile("[^']")

const (
	IO_NUMBER lexer.TokenType = iota + 1 // go-lexer has predefined -1 and 0
	TOKEN
	AND
	AND_IF
	OPENPAREN
	CLOSEPAREN
	SEMI
	DSEMI
	NEWLINE
	OR
	OR_IF
	LESS
	GREAT
	CLOBBER
	DLESS
	DGREAT
	LESSAND
	GREATAND
	DLESSDASH
	LESSGREAT
)

var operators = map[string]lexer.TokenType{
	"&":   AND,
	"&&":  AND_IF,
	"(":   OPENPAREN,
	")":   CLOSEPAREN,
	";":   SEMI,
	";;":  DSEMI,
	"\n":  NEWLINE,
	"|":   OR,
	"||":  OR_IF,
	"<":   LESS,
	">":   GREAT,
	">|":  CLOBBER,
	"<<":  DLESS,
	">>":  DGREAT,
	"<&":  LESSAND,
	">&":  GREATAND,
	"<<-": DLESSDASH,
	"<>":  LESSGREAT,
}

// 2.10.1 - Helper function to detect IO_NUMBERs vs regular TOKENs when an operator begins.
func resolve(current string, delimiter rune) lexer.TokenType {
	if _, err := strconv.Atoi(current); err == nil && (delimiter == '<' || delimiter == '>') {
		// 2.10.1 - Rule 2
		return IO_NUMBER
	} else {
		// 2.10.1 - Rule 3
		return TOKEN
	}
}

// 2.2.1 - Helper function to handle escaping.
func eatEscape(l *lexer.L) {
	l.Next()
	l.IgnoreCharacter()
	r := l.Next()

	if r == '\n' {
		l.IgnoreCharacter()
	}
}

// Handle lexing of regular input.
func lexDelimiting(l *lexer.L) lexer.StateFunc {
	for {
		current := l.Current()
		r := l.Peek()
		switch r {
		// 2.3 - Rule 1
		case -1:
			if len(current) > 0 {
				l.Emit(TOKEN)
			}

			l.Next()
			l.Emit(lexer.EOFToken)
		// 2.3 - Rule 4
		case '\\':
			// 2.2.1
			eatEscape(l)
		case '\'':
			// 2.2.2
			l.Next()
			l.TakeManyPattern(stringLiteralPattern)
			l.Next()
		case '"':
			// 2.2.3
			l.StateRecord.Push(lexDelimiting)
			return lexString
		// 2.3 - Rule 5
		case '$', '`':
			break
		// 2.3 - Rule 6
		case '&', '(', ')', ';', '|', '<', '>', '\n':
			if len(current) > 0 {
				l.Emit(resolve(current, r))
			}

			l.StateRecord.Push(lexDelimiting)
			return lexOperator
		// 2.3 - Rule 7
		case ' ':
			l.Emit(TOKEN)
			l.Next()
			l.IgnoreCharacter()
		// 2.3 - Rules 8 & 9
		case '#':
			if len(current) > 0 {
				l.Next()
			} else {
				l.TakeManyPattern(commentPattern)
			}
		// 2.3 - Rule 10
		default:
			l.Next()
		}
	}
}

// 2.2.3 - Handle lexing of double-quoted strings.
func lexString(l *lexer.L) lexer.StateFunc {
	l.Next()

	for {
		r := l.Peek()

		switch r {
		case '$':
			// TODO: Parameter Expansion
		case '`':
			// TODO: Command substitution
		case '\\':
			next := l.PeekMany(2)
			if next == '$' || next == '`' || next == '"' || next == '\\' || next == '\n' {
				eatEscape(l)
			}
		case '"':
			return l.StateRecord.Pop()
		}

		l.Next()
	}
}

// 2.3 - Rules 2 & 3 - Handle lexing of operator tokens.
func lexOperator(l *lexer.L) lexer.StateFunc {
	l.Next()

	for {
		current := l.Current()
		possibleOp := current + string(l.Peek())

		if _, ok := operators[possibleOp]; !ok {
			// 2.10.1 - Rule 1
			l.Emit(operators[current])
			return l.StateRecord.Pop()
		}

		l.Next()
	}
}