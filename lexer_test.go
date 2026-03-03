package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLexer_OneOpen(t *testing.T) {

	tests := []struct {
		query    string
		expected Op
	}{
		{query: `true`, expected: OpBool},
		{query: `tRue`, expected: OpBool},
		{query: `fALse`, expected: OpBool},
		{query: `7`, expected: OpNumberInt},
		{query: `-9`, expected: OpNumberInt},
		{query: `4.2`, expected: OpNumberFloat},
		{query: `0.9`, expected: OpNumberFloat},
		{query: `-0.9`, expected: OpNumberFloat},
		{query: `"false"`, expected: OpString},
		{query: `Or`, expected: OpOr},
		{query: `aND`, expected: OpAnd},
		{query: ` noT `, expected: OpNot},
		{query: ` = `, expected: OpEq},
		{query: ` != `, expected: OpNeq},
		{query: ` < `, expected: OpLt},
		{query: `<=`, expected: OpLe},
		{query: ` > `, expected: OpGt},
		{query: `>=`, expected: OpGe},
		{query: `(`, expected: OpLParen},
		{query: `)`, expected: OpRParen},
		{query: ` , `, expected: OpComma},
		{query: `betWeen`, expected: OpBetween},
		{query: `In`, expected: OpIn},
		{query: ` - `, expected: OpUndefined},

		{query: `startswith`, expected: OpIdent},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			lex := lexer{input: tt.query, pos: 0}
			lexerOpen := lex.nextToken().Op
			assert.Equal(
				t,
				tt.expected,
				lexerOpen,
				fmt.Sprintf("%s != %s", tt.expected, lexerOpen),
			)
		})
	}
}

func TestLexer_ManyOpen(t *testing.T) {

	tests := []struct {
		query    string
		expected []Op
	}{
		{query: `ok = true`, expected: []Op{
			OpIdent,
			OpEq,
			OpBool,
		}},
		{query: `num = -5`, expected: []Op{
			OpIdent,
			OpEq,
			OpNumberInt,
		}},
		{query: `num = -5.3`, expected: []Op{
			OpIdent,
			OpEq,
			OpNumberFloat,
		}},
		{query: `float32(-5)`, expected: []Op{
			OpIdent,
			OpLParen,
			OpNumberInt,
			OpRParen,
		}},
		{query: `not(ok = true)`, expected: []Op{
			OpNot,
			OpLParen,
			OpIdent,
			OpEq,
			OpBool,
			OpRParen,
		}},
		{query: `ok != true`, expected: []Op{
			OpIdent,
			OpNeq,
			OpBool,
		}},
		{query: `name = "Inge" and age = 3`, expected: []Op{
			OpIdent,
			OpEq,
			OpString,
			OpAnd,
			OpIdent,
			OpEq,
			OpNumberInt,
		}},
		{query: `name="Inge" or age=3`, expected: []Op{
			OpIdent,
			OpEq,
			OpString,
			OpOr,
			OpIdent,
			OpEq,
			OpNumberInt,
		}},

		{query: `name startswith "Ma"`, expected: []Op{
			OpIdent,
			OpIdent,
			OpString,
		}},
		{query: `name between("a", "x")`, expected: []Op{
			OpIdent,
			OpBetween,
			OpLParen,
			OpString,
			OpComma,
			OpString,
			OpRParen,
		}},
		{query: `name IN("a", "x")`, expected: []Op{
			OpIdent,
			OpIn,
			OpLParen,
			OpString,
			OpComma,
			OpString,
			OpRParen,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			lex := lexer{input: tt.query, pos: 0}
			for _, Open := range tt.expected {
				lexerOpen := lex.nextToken().Op
				assert.Equal(
					t,
					Open,
					lexerOpen,
					fmt.Sprintf("%s != %s", Open, lexerOpen),
				)
			}
		})
	}
}

func TestLexer_Invalid(t *testing.T) {

	tests := []struct {
		query    string
		expected []Op
	}{
		{query: `3.3.1`, expected: []Op{
			OpNumberFloat,
			OpEOF,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			lex := lexer{input: tt.query, pos: 0}
			for _, Open := range tt.expected {
				lexerOpen := lex.nextToken().Op
				assert.Equal(
					t,
					Open,
					lexerOpen,
					fmt.Sprintf("%s != %s", Open, lexerOpen),
				)
			}
		})
	}
}
