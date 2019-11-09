package main

import (
	"fmt"
	"github.com/ZadenRB/go-lexer"
)

type ASTNode struct {
	Value *lexer.Token
	Left *ASTNode
	Right *ASTNode
}

var reserved_words = map[string]lexer.TokenType {
	"if":IF,
	"then":THEN,
	"else":ELSE,
	"elif":ELIF,
	"fi":FI,
	"do":DO,
	"done":DONE,
	"case":CASE,
	"esac":ESAC,
	"while":WHILE,
	"until":UNTIL,
	"for":FOR,
	"{":LBRACE,
	"}":RBRACE,
	"!":BANG,
	"in":IN,
}

func Parse(tokens chan lexer.Token) ASTNode {
	var AST ASTNode
	tok := <-tokens
	for tok.Type != lexer.EOFToken {
		var err error
		AST, err = parseProgram(tok)
		if err != nil {
			fmt.Println(err)
			break
		}
		tok = <-tokens
	}
	return AST
}

func parseProgram(tok lexer.Token) (ASTNode, error) {
	
}

func expect(tokType lexer.TokenType) *ASTNode {

}

