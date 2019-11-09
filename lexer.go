package main

import (
	"github.com/ZadenRB/go-lexer"
	"regexp"
	"strconv"
)

const (
	IO_NUMBER lexer.TokenType = iota
	TOKEN
	WORD
	ASSIGNMENT_WORD
	NAME
	// Reserved Words
	IF
	THEN
	ELSE
	ELIF
	FI
	DO
	DONE
	CASE
	ESAC
	WHILE
	UNTIL
	FOR
	LBRACE
	RBRACE
	BANG
	IN
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

var operators = map[string]lexer.TokenType {
	"&":AND,
	"&&":AND_IF,
	"(":OPENPAREN,
	")":CLOSEPAREN,
	";":SEMI,
	";;":DSEMI,
	string('\n'):NEWLINE,
	"|":OR,
	"||":OR_IF,
	"<":LESS,
	">":GREAT,
	">|":CLOBBER,
	"<<":DLESS,
	">>":DGREAT,
	"<&":LESSAND,
	">&":GREATAND,
	"<<-":DLESSDASH,
	"<>":LESSGREAT,
}

func resolve(current string, delimChar string) lexer.TokenType {
	if val, ok := operators[current]; ok {
		return val
	} else if _, err := strconv.Atoi(current); err == nil && delimChar == ">" || delimChar == "<"{
		return IO_NUMBER
	} else {
		return TOKEN
	}
}

// State functions
func lexDelimitation(l *lexer.L) lexer.StateFunc {
	for {
		r := l.Next() //Rule 8 & 10 2.3
		if r == -1 { //Rule 1 2.3
			if len(l.Current()) > 0 {
				l.Emit(resolve(l.Current(), "EOF"))
			}
			l.Emit(lexer.EOFToken)
		} else if matches, _ := regexp.MatchString("[\\\"']", string(r)); matches { //Rule 4 2.3
			switch r {
			case '\\':
				l.Backup()
				l.StateRecord.Push(lexDelimitation)
				return lexEscape
			case '"':
				l.StateRecord.Push(lexDelimitation)
				return lexString
			case '\'':
				l.StateRecord.Push(lexDelimitation)
				return lexLiteralString
			}
		} else if matches, _ := regexp.MatchString("[&();|<>]", string(r)); matches { //Rule 6 2.3
			current := l.Current()
			delimChar := current[len(current) - 1:]
			l.Backup()
			if current := l.Current(); len(current) > 0 {
				l.Emit(resolve(current, delimChar))
			}
			l.Next()
			l.StateRecord.Push(lexDelimitation)
			return lexOperator
		} else if r == ' ' { //Rule 7 2.3
			l.Backup()
			l.Emit(resolve(l.Current(), " "))
			l.Next()
			l.IgnoreCharacter()
		} else if r == '#' { //Rule 9 2.3
			for {
				r = l.Peek()
				if r == '\n' {
					l.Ignore()
					break
				}
				l.Next()
			}
		}
	}
}

func lexOperator(l *lexer.L) lexer.StateFunc { //Rules 2 & 3 2.3
	for {
		r := l.Peek()
		current := l.Current()
		possibleOp := current + string(r)
		if _, ok := operators[possibleOp]; !ok {
			l.Emit(operators[current])
			l.Next()
			return l.StateRecord.Pop()
		}
		l.Next()
	}
}

func lexEscape(l *lexer.L) lexer.StateFunc {
	l.IgnoreCharacter()
	l.Next()
	return l.StateRecord.Pop()
}

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
		case '\'':
			l.StateRecord.Push(lexString)
			return lexLiteralString
		case '"':
			return l.StateRecord.Pop()
		}
	}
}

func lexLiteralString(l *lexer.L) lexer.StateFunc {
	for {
		r := l.Next()
		switch r {
		case '\'': return l.StateRecord.Pop()
		}
	}
}
