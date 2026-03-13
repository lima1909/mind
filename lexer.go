package mind

import "fmt"

type Op int

const (
	// Mask the high 16 bits for the category
	opCategoryMaskOp Op = 0xFFFF0000

	// Categories moved to the high 16 bits
	opRelational Op = 0x00010000
	opLogical    Op = 0x00020000
	opDatatype   Op = 0x00030000
	opStructural Op = 0x00040000
)

// Structural & Literals
const (
	OpUndefined Op = opStructural | iota
	OpEOF
	OpLParen
	OpRParen
	OpIdent
	OpComma
)

// Datatypes
const (
	OpString Op = opDatatype | iota
	OpNumberInt
	OpNumberFloat
	OpBool
)

// Logical
const (
	OpAnd Op = opLogical | iota
	OpOr
	OpNot
)

// Relation (Payloads can now safely use bits 0 through 15)
const (
	OpEq      Op = opRelational | (1 << 0)
	OpNeq        = opRelational | (1 << 1)
	OpLt         = opRelational | (1 << 2)
	OpLe         = opRelational | (1 << 3)
	OpGt         = opRelational | (1 << 4)
	OpGe         = opRelational | (1 << 5)
	OpIn         = opRelational | (1 << 6)
	OpBetween    = opRelational | (1 << 7)
)

func (o Op) IsRelational() bool { return o&opCategoryMaskOp == opRelational }
func (o Op) IsLogical() bool    { return o&opCategoryMaskOp == opLogical }
func (o Op) IsDatatype() bool   { return o&opCategoryMaskOp == opDatatype }
func (o Op) IsStructural() bool { return o&opCategoryMaskOp == opStructural }

// Code returns just the payload (the ID or the bit-flags)
func (o Op) Code() uint16 { return uint16(o & 0xFFFF) }

func (o Op) String() string {
	switch o {
	case OpUndefined:
		return "UNDEFINED"
	case OpEOF:
		return "EOF"
	case OpIdent:
		return "IDENT"
	case OpString:
		return "STRING"
	case OpNumberInt:
		return "NUMBER-INT"
	case OpNumberFloat:
		return "NUMBER-FLOAT"
	case OpBool:
		return "BOOL"
	case OpComma:
		return ","
	case OpEq:
		return "="
	case OpNeq:
		return "!="
	case OpLt:
		return "<"
	case OpLe:
		return "<="
	case OpGt:
		return ">"
	case OpGe:
		return ">="
	case OpIn:
		return "IN"
	case OpBetween:
		return "BETWEEN"
	case OpAnd:
		return "AND"
	case OpOr:
		return "OR"
	case OpNot:
		return "NOT"
	case OpLParen:
		return "("
	case OpRParen:
		return ")"
	default:
		return fmt.Sprintf("UNKNOWN: %d", o)
	}
}

type token struct {
	Start int
	End   int
	Op    Op
}

type lexer struct {
	input string
	pos   int
}

func (l *lexer) nextToken() token {
	// skip whitespace
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.pos++
			continue
		}
		break
	}

	if l.pos >= len(l.input) {
		return token{Op: OpEOF, Start: l.pos, End: l.pos}
	}

	ch := l.input[l.pos]

	switch {
	case ch == '=':
		start := l.pos
		l.pos++
		return token{Op: OpEq, Start: start, End: l.pos}

	case ch == '!':
		start := l.pos
		// Check if the next byte exists and is '='
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2 // Consume both '!' and '='
			return token{Op: OpNeq, Start: start, End: l.pos}
		}
		// Optional: Handle a lone '!' if you want a NOT operator later
		l.pos++
	case ch == '<':
		start := l.pos
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return token{Op: OpLe, Start: start, End: l.pos}
		}
		l.pos++
		return token{Op: OpLt, Start: start, End: l.pos}
	case ch == '>':
		start := l.pos
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.pos += 2
			return token{Op: OpGe, Start: start, End: l.pos}
		}
		l.pos++
		return token{Op: OpGt, Start: start, End: l.pos}
	case (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_':
		return l.readKeyword()
	case ch == '"', ch == '\'':
		return l.readString(ch)
	case (ch >= '0' && ch <= '9') || ch == '-':
		return l.readNumber()
	case ch == '(':
		start := l.pos
		l.pos++
		return token{Op: OpLParen, Start: start, End: l.pos}
	case ch == ')':
		start := l.pos
		l.pos++
		return token{Op: OpRParen, Start: start, End: l.pos}
	case ch == ',':
		start := l.pos
		l.pos++
		return token{Op: OpComma, Start: start, End: l.pos}
	}

	l.pos++
	return token{Op: OpEOF, Start: l.pos, End: l.pos}
}

// readIdentOrKeyword checks if the word is AND / OR without allocating memory
// Keywords are:
// - bool: true, false
// - Logical: or, and, not
// - ident: fieldname
// - operation: between
func (l *lexer) readKeyword() token {
	start := l.pos
	// read while are there letters, numbers or _
	// it starts with a letter
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			l.pos++
		} else {
			break
		}
	}

	length := l.pos - start
	b := l.input[start:]

	// evaluate the founded string
	// Fast Keyword & Boolean Checks
	switch length {
	case 2:
		// OR
		if (b[0] == 'o' || b[0] == 'O') &&
			(b[1] == 'r' || b[1] == 'R') {
			return token{Op: OpOr, Start: start, End: l.pos}
		}
		// IN
		if (b[0] == 'i' || b[0] == 'I') &&
			(b[1] == 'n' || b[1] == 'N') {
			return token{Op: OpIn, Start: start, End: l.pos}
		}
	case 3:
		// AND
		if (b[0] == 'a' || b[0] == 'A') &&
			(b[1] == 'n' || b[1] == 'N') &&
			(b[2] == 'd' || b[2] == 'D') {
			return token{Op: OpAnd, Start: start, End: l.pos}
		}
		// NOT
		if (b[0] == 'n' || b[0] == 'N') &&
			(b[1] == 'o' || b[1] == 'O') &&
			(b[2] == 't' || b[2] == 'T') {
			return token{Op: OpNot, Start: start, End: l.pos}
		}
	case 4:
		// TRUE
		if (b[0] == 't' || b[0] == 'T') &&
			(b[1] == 'r' || b[1] == 'R') &&
			(b[2] == 'u' || b[2] == 'U') &&
			(b[3] == 'e' || b[3] == 'E') {
			return token{Op: OpBool, Start: start, End: l.pos}
		}
	case 5:
		// FALSE
		if (b[0] == 'f' || b[0] == 'F') &&
			(b[1] == 'a' || b[1] == 'A') &&
			(b[2] == 'l' || b[2] == 'L') &&
			(b[3] == 's' || b[3] == 'S') &&
			(b[4] == 'e' || b[4] == 'E') {
			return token{Op: OpBool, Start: start, End: l.pos}
		}
	case 7:
		// BETWEEN
		if (b[0] == 'b' || b[0] == 'B') &&
			(b[1] == 'e' || b[1] == 'E') &&
			(b[2] == 't' || b[2] == 'T') &&
			(b[3] == 'w' || b[3] == 'W') &&
			(b[4] == 'e' || b[4] == 'E') &&
			(b[5] == 'e' || b[5] == 'E') &&
			(b[6] == 'n' || b[6] == 'N') {
			return token{Op: OpBetween, Start: start, End: l.pos}
		}
	}

	// If it didn't match any of the keywords, it's just a normal identifier
	// return token{Type: tokIdent, Start: start, End: l.pos}
	return token{Op: OpIdent, Start: start, End: l.pos}
}

func (l *lexer) readNumber() token {
	input := l.input
	inputLen := len(input)
	start := l.pos
	p := l.pos

	hasDot := false

	if p < inputLen && input[p] == '-' {
		p++
	}

	digitsFound := false

	for p < inputLen {
		ch := input[p]

		if ch >= '0' && ch <= '9' {
			digitsFound = true
			p++
		} else if ch == '.' && !hasDot {
			hasDot = true
			p++
		} else {
			break
		}
	}
	l.pos = p

	// if no digits were found (e.g., input was just "-" or "."),
	if !digitsFound {
		return token{Op: OpUndefined, Start: start, End: l.pos}
	}

	var op Op
	switch {
	case hasDot:
		op = OpNumberFloat
	default:
		op = OpNumberInt
	}

	return token{Op: op, Start: start, End: l.pos}
}

func (l *lexer) readString(quote byte) token {
	l.pos++ // Skip open quote
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != quote {
		l.pos++
	}
	end := l.pos
	if l.pos < len(l.input) {
		l.pos++ // Skip close quote
	}
	return token{Op: OpString, Start: start, End: end}
}
