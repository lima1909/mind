package main

import (
	"fmt"
	"strings"
)

type ExprKind uint8

const (
	ExprTerm ExprKind = iota
	ExprOr
	ExprAnd
	ExprAndNot
	ExprNot
)

type Expr interface{ kind() ExprKind }

type BinaryExpr struct {
	Op    ExprKind // exprOr, exprAnd, exprAndNot
	Left  Expr
	Right Expr
}

func (e BinaryExpr) kind() ExprKind { return e.Op }

type NotExpr struct{ Child Expr }

func (e NotExpr) kind() ExprKind { return ExprNot }

type TermExpr struct {
	Field string
	Op    Op
	Value any
}

func (e TermExpr) kind() ExprKind { return ExprTerm }

type TermManyExpr struct {
	Field            string
	Op               Op
	Values           []any
	MinIncl, MaxIncl bool
}

func (e TermManyExpr) kind() ExprKind { return ExprTerm }

// Parser impl starts
type parser struct {
	input string
	lex   lexer
	cur   token
}

func optimize(e Expr) Expr {
	if e == nil {
		return nil
	}

	switch n := e.(type) {
	case BinaryExpr:
		left := optimize(n.Left)
		right := optimize(n.Right)

		if n.Op == ExprAnd {
			// RULE: And(A, Not(B)) -> AndNot(A, B)
			if notNode, ok := right.(NotExpr); ok {
				return BinaryExpr{Op: ExprAndNot, Left: left, Right: notNode.Child}
			}
			// RULE: And(A, Not(B)) -> AndNot(A, B)
			if notNode, ok := left.(NotExpr); ok {
				return BinaryExpr{Op: ExprAndNot, Left: right, Right: notNode.Child}
			}

			// RULE: And(A > X, B < Y) -> BETWEEN(A, B)
			if lt, okL := left.(TermExpr); okL {
				if rt, okR := right.(TermExpr); okR {
					if lt.Field == rt.Field {
						var min, max any
						var minInc, maxInc bool

						// Identify Lower Bound
						if lt.Op == OpGt || lt.Op == OpGe {
							min, minInc = lt.Value, (lt.Op == OpGe)
						} else if rt.Op == OpGt || rt.Op == OpGe {
							min, minInc = rt.Value, (rt.Op == OpGe)
						}

						// Identify Upper Bound
						if lt.Op == OpLt || lt.Op == OpLe {
							max, maxInc = lt.Value, (lt.Op == OpLe)
						} else if rt.Op == OpLt || rt.Op == OpLe {
							max, maxInc = rt.Value, (rt.Op == OpLe)
						}

						// If we found both a min and a max, we have a BETWEEN
						if min != nil && max != nil {
							return TermManyExpr{
								Field:   lt.Field,
								Op:      OpBetween,
								Values:  []any{min, max},
								MinIncl: minInc, MaxIncl: maxInc,
							}
						}

					}
				}
			}
		}
		return BinaryExpr{Op: n.Op, Left: left, Right: right}

	case NotExpr:
		child := optimize(n.Child)
		switch c := child.(type) {
		// RULE: Not(Not(A)) -> A (Double Negative)
		case NotExpr:
			return optimize(c.Child)
		case TermExpr:
			switch c.Op {
			// I'm not sure, that this is faster
			// RULE: NOT (A = B)  -->  A != B
			//  case OpEq:
			//      return TermExpr{Field: c.Field, Op: OpNeq, Value: c.Value}
			// RULE: NOT (A != B)  -->  A = B
			case OpNeq:
				return TermExpr{Field: c.Field, Op: OpEq, Value: c.Value}
			// RULE: NOT (A > B) --> A <= B
			case OpGt:
				return TermExpr{Field: c.Field, Op: OpLe, Value: c.Value}
			// RULE: NOT (A >= B) --> A < B
			case OpGe:
				return TermExpr{Field: c.Field, Op: OpLt, Value: c.Value}
			// RULE: NOT (A < B) --> A >= B
			case OpLt:
				return TermExpr{Field: c.Field, Op: OpGe, Value: c.Value}
			// RULE: NOT (A <= B) --> A > B
			case OpLe:
				return TermExpr{Field: c.Field, Op: OpGt, Value: c.Value}

			default:
				//  no otimizations
				return n
			}
		default:
			//  no otimizations
			return n

		}
	default:
		return e
	}
}

func compile(e Expr) Query32 {
	switch n := e.(type) {
	case TermExpr:
		return match[uint32](n.Field, n.Op, n.Value)

	case NotExpr:
		return Not(compile(n.Child))

	case BinaryExpr:
		left := compile(n.Left)
		right := compile(n.Right)

		switch n.Op {
		case ExprAnd:
			return And(left, right)
		case ExprOr:
			return Or(left, right)
		case ExprAndNot:
			return AndNot(left, right)
		}
	case TermManyExpr:
		return matchMany[uint32](n.Field, n.Op, n.Values...)
	}

	panic(fmt.Sprintf("NOT supported Expression in compile: %T", e))
}

func Parse(input string) (Query32, error) {
	p := parser{input: input, lex: lexer{input: input, pos: 0}}
	p.next()
	ast, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.cur.Op != OpEOF {
		return nil, p.unexpectedWithMsg("unexpected end of the input")
	}

	// fmt.Println(ast)
	optAst := optimize(ast)
	// fmt.Println(optAst)

	query := compile(optAst)

	return query, nil
}

//go:inline
func (p *parser) next() { p.cur = p.lex.nextToken() }

func (p *parser) parseOr() (Expr, error) {
	// the rule: AND before OR
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.cur.Op == OpOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: ExprOr, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseCondition()
	if err != nil {
		return nil, err
	}

	for p.cur.Op == OpAnd {
		p.next()
		right, err := p.parseCondition()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: ExprAnd, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseCondition() (Expr, error) {
	if p.cur.Op == OpNot {
		p.next() // consume 'NOT'
		// Recursively parse the expression that follows
		expr, err := p.parseCondition()
		if err != nil {
			return nil, err
		}
		return NotExpr{Child: expr}, nil
	}

	if p.cur.Op == OpLParen {
		p.next()
		expr, err := p.parseOr() // Back to the top of the precedence chain
		if err != nil {
			return nil, err
		}
		if p.cur.Op != OpRParen {
			return nil, p.unexpected(OpRParen)
		}
		p.next()
		return expr, nil
	}

	if p.cur.Op != OpIdent {
		return nil, p.unexpected(OpIdent)
	}
	field := p.input[p.cur.Start:p.cur.End]
	p.next()

	tokenOp := p.cur.Op
	p.next()
	switch tokenOp {
	case OpNeq:
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		return NotExpr{Child: TermExpr{Field: field, Op: OpEq, Value: val}}, nil
	case OpLt, OpLe, OpGt, OpGe, OpEq:
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		return TermExpr{Field: field, Op: tokenOp, Value: val}, nil
	case OpBetween:
		values, err := p.parseValueList()
		if err != nil {
			return nil, err
		}
		//TODO: check there are 2 values (from,to)
		return TermManyExpr{Field: field, Op: OpBetween, Values: values}, nil
	case OpIn:
		values, err := p.parseValueList()
		if err != nil {
			return nil, err
		}
		return TermManyExpr{Field: field, Op: OpIn, Values: values}, nil
	// case tokIdent:
	// maybe relations like startswith
	default:
		return nil, p.unexpectedWithMsg("missing relation like: =, !=, <, ...")
	}
}

func (p *parser) parseValueList() ([]any, error) {
	if p.cur.Op != OpLParen {
		return nil, p.unexpected(OpLParen)
	}
	p.next()

	expectedOpValue := OpEOF
	values := make([]any, 0, 10)

	for {
		// all list values should have the same type
		if expectedOpValue == OpEOF {
			expectedOpValue = p.cur.Op
		} else if expectedOpValue != p.cur.Op {
			return nil, p.unexpected(expectedOpValue)
		}

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, val)

		if p.cur.Op == OpRParen {
			p.next()
			break
		}

		if p.cur.Op != OpComma {
			return nil, p.unexpected(OpComma)
		}
		p.next()
	}

	return values, nil
}

func (p *parser) parseValue() (any, error) {
	var val any
	switch p.cur.Op {
	case OpString:
		val = p.input[p.cur.Start:p.cur.End]
	case OpNumberInt:
		val = parseInt(p.input[p.cur.Start:p.cur.End])
	case OpNumberFloat:
		val = parseFloat(p.input[p.cur.Start:p.cur.End])
	case OpBool:
		val = parseBool(p.input[p.cur.Start:p.cur.End])
	default:
		return nil, p.unexpectedWithMsg("missing value like: string, number, bool")
	}
	p.next()
	return val, nil
}

func parseBool(s string) bool {
	switch len(s) {
	case 4:
		if (s[0] == 't' || s[0] == 'T') &&
			(s[1] == 'r' || s[1] == 'R') &&
			(s[2] == 'u' || s[2] == 'U') &&
			(s[3] == 'e' || s[3] == 'E') {
			return true
		}
	case 5:
		if (s[0] == 'f' || s[0] == 'F') &&
			(s[1] == 'a' || s[1] == 'A') &&
			(s[2] == 'l' || s[2] == 'L') &&
			(s[3] == 's' || s[3] == 'S') &&
			(s[4] == 'e' || s[4] == 'E') {
			return false
		}
	}

	return false
}

//go:inline
func parseUint(s string) uint64 {
	var n uint64

	// BCE (Bounds Check Elimination):
	// By checking the length once at the start, the Go compiler
	// removes all bounds checks inside the loop.
	_ = s[len(s)-1]

	for i := 0; i < len(s); i++ {
		// math trick: n * 10 is compiled into (n << 3) + (n << 1)
		// which is much faster than the MUL instruction on some CPUs.
		n = n*10 + uint64(s[i]-'0')
	}

	return n
}

func parseInt(s string) int64 {
	if s[0] == '-' {
		s = s[1:]
		return -int64(parseUint(s))
	}

	return int64(parseUint(s))
}

func parseFloat(s string) float64 {
	neg := false
	if len(s) > 0 && s[0] == '-' {
		neg = true
		s = s[1:]
	}

	var mantissa uint64
	dotPos := -1

	_ = s[len(s)-1] // BCE

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			dotPos = i
			continue
		}
		mantissa = mantissa*10 + uint64(c-'0')
	}

	var result float64
	if dotPos < 0 {
		// No dot found, it's just an integer
		result = float64(mantissa)
	} else {
		// Calculate how many fractional digits we have
		fracDigits := len(s) - 1 - dotPos

		// If it's within our precomputed range, use the blazing fast array lookup
		if fracDigits < len(powersOf10) {
			result = float64(mantissa) / powersOf10[fracDigits]
			// } else {
			// Fallback: If it has >18 fractional digits, the fast path fails.
			// Let the standard library handle extreme precision.
			// In fali, you might want to call strconv.ParseFloat here.
		}
	}

	if neg {
		return -result
	}
	return result
}

// pre-computed powers of 10 to avoid expensive math.Pow() calls
var powersOf10 = [...]float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18,
}

// --- parse error ---
func (p *parser) unexpected(expected Op) error {
	return UnexpectedTokenError{input: p.input, token: p.cur, expected: expected}
}

func (p *parser) unexpectedWithMsg(msg string) error {
	return UnexpectedTokenError{input: p.input, token: p.cur, msg: msg, expected: OpUndefined}
}

type UnexpectedTokenError struct {
	input    string
	msg      string
	token    token
	expected Op
}

func (e UnexpectedTokenError) Error() string {
	var msg string
	if e.msg != "" {
		msg = fmt.Sprintf("%s at position %d", e.msg, e.token.Start)
	} else if e.expected == OpUndefined {
		msg = fmt.Sprintf("unexpected token %q at position %d", e.token.Op, e.token.Start)
	} else {
		msg = fmt.Sprintf("expected %q, got %q at position %d", e.expected, e.token.Op, e.token.Start)
	}

	if e.input == "" {
		return msg
	}

	// Build a visual pointer: show the input and a caret line under the error position.
	start := e.token.Start
	end := e.token.End
	if start > len(e.input) {
		start = len(e.input)
	}
	if end > len(e.input) {
		end = len(e.input)
	}

	caretLen := end - start
	if caretLen < 1 {
		caretLen = 1
	}

	return fmt.Sprintf("%s\n  %s\n  %s%s",
		msg,
		e.input,
		strings.Repeat(" ", start),
		strings.Repeat("^", caretLen),
	)
}
