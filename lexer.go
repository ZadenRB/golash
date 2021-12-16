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

const (
	IO_NUMBER lexer.TokenType = iota + 1 // go-lexer has predefined -1 and 0
	TOKEN
	// Control Operators
	AND
	AND_IF
	OPENPAREN
	CLOSEPAREN
	SEMI
	DSEMI
	NEWLINE
	OR
	OR_IF
	// Redirection Operators
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

func resolve(current string) lexer.TokenType {
	lastChar := current[len(current)-1:]
	if _, err := strconv.Atoi(current[:len(current)-1]); err == nil && (lastChar == ">" || lastChar == "<") {
		// 2.10.1 - Rule 2
		return IO_NUMBER
	} else {
		return TOKEN
	}
}

// State functions
func lexDelimiting(l *lexer.L) lexer.StateFunc {
	for {
		current := l.Current()
		r := l.Peek()
		switch r {
		// 2.3 - Rule 1
		case -1:
			if len(current) > 0 {
				l.Emit(resolve(current))
			}

			l.Emit(lexer.EOFToken)
		// 2.3 - Rule 4
		case '\\':
			l.StateRecord.Push(lexDelimiting)
			return lexEscape
		case '\'':
			l.StateRecord.Push(lexDelimiting)
			return lexLiteralString
		case '"':
			l.StateRecord.Push(lexDelimiting)
			return lexString
		// 2.3 - Rule 5
		case '$', '`':
			break
		// 2.3 - Rule 6
		case '&', '(', ')', ';', '|', '<', '>', '\n':
			if len(current) > 0 {
				l.Emit(resolve(current))
			}

			l.StateRecord.Push(lexDelimiting)
			return lexOperator
		// 2.3 - Rule 7
		case ' ':
			l.Emit(resolve(current))
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

// 2.3 - Rules 2 & 3
func lexOperator(l *lexer.L) lexer.StateFunc {
	for {
		current := l.Current()
		possibleOp := current + string(l.Peek())

		if _, ok := operators[possibleOp]; !ok {
			l.Emit(operators[current])
			return l.StateRecord.Pop()
		}
	}
}

// 2.2.1
func lexEscape(l *lexer.L) lexer.StateFunc {
	l.Next()
	l.IgnoreCharacter()
	r := l.Peek()

	if r == '\n' {
		l.Next()
		l.IgnoreCharacter()
	}

	return l.StateRecord.Pop()
}

// 2.2.2
func lexLiteralString(l *lexer.L) lexer.StateFunc {
	for {
		r := l.Next()

		if r == '\'' {
			return l.StateRecord.Pop()
		}
	}
}

// 2.2.3
func lexString(l *lexer.L) lexer.StateFunc {
	for {
		r := l.Next()

		switch r {
		case '\\':
			next := l.Peek()
			if matches, _ := regexp.MatchString("[$`\"\\\\n]", string(next)); matches {
				l.Backup()
				l.StateRecord.Push(lexString)
				return lexEscape
			}
		case '`':
			// TODO: Command substitution
		case '$':
			// TODO: Parameter Expansion
		case '"':
			return l.StateRecord.Pop()
		}
	}
}
