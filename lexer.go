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

var commentPattern = regexp.MustCompile(`[^\n]`)
var stringLiteralPattern = regexp.MustCompile(`[^']`)
var namePattern = regexp.MustCompile(`[A-Za-z_\d]`)

const (
	IO_NUMBER lexer.TokenType = iota + 1 // go-lexer has predefined -1 and 0
	TOKEN
	AND
	AND_IF
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
// Called with current character being '\\'.
// Exits with current character being whatever followed the '\\'.
func eatEscape(l *lexer.L) {
	l.IgnoreCharacter()
	r := l.Next()

	if r == '\n' {
		l.IgnoreCharacter()
	}
}

// 2.2.2 - Helper function to handle single quoting.
// Called with current character being '\''.
// Exits with current character being '\''.
func eatStringLiteral(l *lexer.L) {
	l.TakeManyPattern(stringLiteralPattern)
	l.Next()
}

// 2.2.3 - Helper function to handle double quoting.
// Called with current character being '"'.
// Exits with current character being '"'.
func eatString(l *lexer.L) {
	ok := true
	for ok {
		switch l.Next() {
		case '$':
			if l.Peek() == '(' {
				if l.PeekMany(2) == '(' {
					eatArithmeticExpansion(l)
				}

				eatDollarCommandSubstitution(l)
			}

			eatParameterExpansion(l)
		case '`':
			eatBacktickCommandSubstitution(l)
		case '\\':
			next := l.Peek()
			if next == '$' || next == '`' || next == '"' || next == '\\' || next == '\n' {
				eatEscape(l)
			}
		case '"':
			ok = false
		}
	}
}

// 2.6.2 - Helper function to handle parameter expansion.
// Called with current character being '$'.
// Exits with current character being '}' or the last character of a name.
func eatParameterExpansion(l *lexer.L) {
	switch l.Peek() {
	case '{':
		braceCount := 0
		for {
			r := l.Next()

			switch r {
			case '{':
				braceCount += 1
			case '}':
				braceCount -= 1
			case '"':
				eatString(l)
			case '\'':
				eatStringLiteral(l)
			case '\\':
				eatEscape(l)
			case '$':
				if l.PeekMany(2) == '(' {
					if l.PeekMany(3) == '(' {
						eatArithmeticExpansion(l)
					}

					eatDollarCommandSubstitution(l)
				}

				eatParameterExpansion(l)
			case '`':
				eatBacktickCommandSubstitution(l)
			}

			if braceCount == 0 {
				break
			}
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '@', '*', '#', '?', '-', '$', '!':
		l.Next()
	default:
		l.TakeManyPattern(namePattern)
	}
}

// 2.6.3 - Helper function to handle '$(...)' command substitution.
// Applies tokenizing rules from 2.3 but doesn't emit tokens (except EOF).
// Called with current character being '$'.
// Exits with current character being ')'.
func eatDollarCommandSubstitution(l *lexer.L) {
	for {
		switch l.Next() {
		case -1:
			l.Backup()
			if len(l.Current()) > 0 {
				l.Emit(TOKEN)
			}

			l.Next()
			l.Emit(lexer.EOFToken)
		case '\\':
			eatEscape(l)
		case '\'':
			eatStringLiteral(l)
		case '"':
			eatString(l)
		case '$':
			if l.Peek() == '(' {
				if l.PeekMany(2) == '(' {
					eatArithmeticExpansion(l)
				}

				eatDollarCommandSubstitution(l)
			}

			eatParameterExpansion(l)
		case '`':
			eatBacktickCommandSubstitution(l)
		case '&', ';', '|', '<', '>', '\n':
			eatOperator(l)
		case '#':
			// If the current token consists only of the #, it is a comment.
			if len(l.Current()) == 1 {
				l.TakeManyPattern(commentPattern)
			}
		}
	}
}

// 2.6.3 - Helper function to handle '`...`' command substitution.
// Called with current character being '`'.
// Exits with current character being '`'.
func eatBacktickCommandSubstitution(l *lexer.L) {
	for {
		switch l.Next() {
		case '"':
			eatString(l)
		case '\'':
			eatStringLiteral(l)
		case '\\':
			next := l.Peek()
			if next == '$' || next == '`' || next == '\\' {
				eatEscape(l)
			}
		case '`':
			return
		}
	}
}

// 2.6.4 - Helper function to handle arithmetic expansion.
// Called with current character being '$'.
// Exits with current character being ')'.
func eatArithmeticExpansion(l *lexer.L) {
	parenCount := 0
	for {
		r := l.Next()

		switch r {
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '$':
			if l.PeekMany(2) == '(' {
				if l.PeekMany(3) == '(' {
					eatArithmeticExpansion(l)
				}

				eatDollarCommandSubstitution(l)
			}

			eatParameterExpansion(l)
		case '`':
			eatBacktickCommandSubstitution(l)
		case '\\':
			next := l.PeekMany(2)
			if next == '$' || next == '`' || next == '"' || next == '\\' || next == '\n' {
				eatEscape(l)
			}
		}

		if parenCount == 0 {
			break
		}
	}
}

// 2.10.1 Rule 1 - Helper function to handle operators without emitting tokens.
// Called with current character being the first character of an operator.
// Exits with the current character being the last character of an operator.
func eatOperator(l *lexer.L) {
	i := 1
	for {
		current := l.Current()
		currentOp := current[len(current)-i:]
		possibleOp := currentOp + string(l.Peek())

		if _, ok := operators[possibleOp]; !ok {
			return
		}

		l.Next()
		i++
	}
}

// 2.3 Rules 1-10 - Handle lexing of regular input.
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
			l.StateRecord.Push(lexDelimiting)

			return lexEscape
		case '\'':
			// 2.2.2
			l.StateRecord.Push(lexDelimiting)

			return lexStringLiteral
		case '"':
			// 2.2.3
			l.StateRecord.Push(lexDelimiting)

			return lexString
		// 2.3 - Rule 5
		case '$':
			l.StateRecord.Push(lexDelimiting)

			if l.PeekMany(2) == '(' {
				if l.PeekMany(3) == '(' {
					return lexArithmeticExpansion
				}

				return lexDollarCommandSubstitution
			}

			return lexParameterExpansion
		case '`':
			return lexBacktickCommandSubstitution
		// 2.3 - Rule 6
		case '&', ';', '|', '<', '>', '\n':
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

// 2.2.1 - Handle lexing of escaped characters.
// Called with the next character being '\\'.
// Exits with the current character being whatever followed the '\\'.
func lexEscape(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatEscape(l)

	return l.StateRecord.Pop()
}

// 2.2.2 - Handle lexing of single-quoted strings.
// Called with the next character being '\''.
// Exits with the current character being '\''.
func lexStringLiteral(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatStringLiteral(l)

	return l.StateRecord.Pop()
}

// 2.2.3 - Handle lexing of double-quoted strings.
// Called with the next character being '"'.
// Exits with the current character being '"'.
func lexString(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatString(l)

	return l.StateRecord.Pop()
}

// 2.3 - Rules 2 & 3 - Handle lexing of operator tokens.
// Called with current character being the first character of an operator.
// Exits with the current character being the last character of an operator.
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

// 2.3 - Rule 5 / 2.6.2 - Handle lexing of parameter expansion.
// Called with next character being '$'.
// Exits with the current character being the last character in parameter expansion.
func lexParameterExpansion(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatParameterExpansion(l)

	return l.StateRecord.Pop()
}

// 2.3 - Rule 5 / 2.6.3 - Handle lexing of '$(...)' command substitution
// Called with next character being '$'.
// Exits with the current character being ')'.
func lexDollarCommandSubstitution(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatDollarCommandSubstitution(l)

	return l.StateRecord.Pop()
}

// 2.3 - Rule 5 / 2.6.3 - Handle lexing of '`...`' command substitution.
// Called with next character being '`'.
// Exits with the current character being '`'.
func lexBacktickCommandSubstitution(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatBacktickCommandSubstitution(l)

	return l.StateRecord.Pop()
}

// 2.3 - Rule 5 / 2.6.4 - Handle lexing of arithmetic expansion.
// Called with next character being '$'.
// Exits with the current character being ')'.
func lexArithmeticExpansion(l *lexer.L) lexer.StateFunc {
	l.Next()

	eatArithmeticExpansion(l)

	return l.StateRecord.Pop()
}
