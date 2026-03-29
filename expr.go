package mind

import (
	"fmt"
	"strings"
	"time"
)

type ExprKind uint8

const (
	ExprOr ExprKind = iota
	ExprAnd
	ExprAndNot
)

func (e ExprKind) String() string {
	switch e {
	case ExprOr:
		return " OR "
	case ExprAnd:
		return " AND "
	case ExprAndNot:
		return " ANDNOT "
	default:
		return " "
	}
}

type Tracer struct {
	Expr     Expr
	Duration time.Duration
	Matches  int
	Children []*Tracer
}

func (t *Tracer) Trace(query Query, expr Expr, children ...*Tracer) Query {
	if t == nil {
		return query
	}

	t.Expr = expr
	t.Children = children

	return func(l FilterByName, allIDs *RawIDs32) (*RawIDs32, bool, error) {
		start := time.Now()
		ids, canMutate, err := query(l, allIDs)
		if err != nil {
			return ids, canMutate, err
		}

		t.Duration = time.Since(start)
		t.Matches = ids.Count()

		return ids, canMutate, err
	}
}

func (t *Tracer) Print(indent string, isLast bool) string {
	var sb strings.Builder

	// Choose the correct prefix symbols
	marker := "├── "
	if isLast {
		marker = "└── "
	}

	// Format: Node Description | Duration | Matches
	line := fmt.Sprintf("%s%s%s  [%v] (%d matches)\n",
		indent, marker, t.Expr, t.Duration.Round(time.Nanosecond), t.Matches)
	sb.WriteString(line)

	// Prepare indentation for children
	newIndent := indent
	if isLast {
		newIndent += "    "
	} else {
		newIndent += "│   "
	}

	// Recursively print children
	for i, child := range t.Children {
		lastChild := i == len(t.Children)-1
		sb.WriteString(child.Print(newIndent, lastChild))
	}

	return sb.String()
}

func (t *Tracer) String() string {
	if t == nil {
		return "<nil trace>"
	}
	return t.Print("", true)
}

type Expr interface {
	Equals(Expr) bool
	Compile(*Tracer) Query
	String() string
}

type BinaryExpr struct {
	Ekind ExprKind
	Left  Expr
	Right Expr
}

func (e BinaryExpr) Equals(other Expr) bool {
	o, ok := other.(BinaryExpr)
	if !ok {
		return false
	}

	if e.Ekind != o.Ekind {
		return false
	}

	return e.Left.Equals(o.Left) && e.Right.Equals(o.Right)
}

func (e BinaryExpr) Compile(t *Tracer) Query {
	var leftTracer, rightTracer *Tracer
	if t != nil {
		leftTracer, rightTracer = &Tracer{}, &Tracer{}
	}

	left := e.Left.Compile(leftTracer)
	right := e.Right.Compile(rightTracer)

	var query Query

	switch e.Ekind {
	case ExprOr:
		query = Or(left, right)
	case ExprAnd:
		query = And(left, right)
	case ExprAndNot:
		query = AndNot(left, right)
	default:
		panic(fmt.Sprintf("Not supported BinaryExpr: %v", e))
	}

	return t.Trace(query, e, leftTracer, rightTracer)
}

func (e BinaryExpr) String() string { return fmt.Sprintf("%s%s%s", e.Left, e.Ekind, e.Right) }

type NotExpr struct{ Child Expr }

func (e NotExpr) Equals(other Expr) bool {
	o, ok := other.(NotExpr)
	if !ok {
		return false
	}

	return e.Child.Equals(o.Child)
}

func (e NotExpr) Compile(t *Tracer) Query {
	var childTracer *Tracer
	if t != nil {
		childTracer = &Tracer{}
	}

	return t.Trace(Not(e.Child.Compile(childTracer)), e, childTracer)
}

func (e NotExpr) String() string { return fmt.Sprintf(" NOT(%s) ", e.Child) }

type TermExpr struct {
	Field string
	Op    FilterOp
	Value any
}

func (e TermExpr) Equals(other Expr) bool {
	o, ok := other.(TermExpr)
	if !ok {
		return false
	}

	return e.Field == o.Field && e.Op == o.Op && e.Value == o.Value
}

func (e TermExpr) Compile(t *Tracer) Query { return t.Trace(match(e.Field, e.Op, e.Value), e) }
func (e TermExpr) String() string          { return fmt.Sprintf("%s %s %v", e.Field, e.Op, e.Value) }

type TermManyExpr struct {
	Field  string
	Op     FilterOp
	Values []any
}

func (e TermManyExpr) Equals(other Expr) bool {
	o, ok := other.(TermManyExpr)
	if !ok {
		return false
	}

	if len(e.Values) != len(o.Values) {
		return false
	}

	for i, ev := range e.Values {
		if ev != o.Values[i] {
			return false
		}
	}

	return e.Field == o.Field && e.Op == o.Op
}

func (e TermManyExpr) Compile(t *Tracer) Query {
	return t.Trace(matchMany(e.Field, e.Op, e.Values...), e)
}
func (e TermManyExpr) String() string { return fmt.Sprintf("%s %s %v", e.Field, e.Op, e.Values) }

// FalseExpr represents a condition that is always false.
// like: A > 10 AND A < 5
type FalseExpr struct{}

func (e FalseExpr) Equals(other Expr) bool  { _, ok := other.(FalseExpr); return ok }
func (e FalseExpr) Compile(t *Tracer) Query { return t.Trace(empty(), e) }
func (e FalseExpr) String() string          { return "FALSE" }

// TrueExpr represents a condition that is always true.
type TrueExpr struct{}

func (e TrueExpr) Equals(other Expr) bool  { _, ok := other.(TrueExpr); return ok }
func (e TrueExpr) Compile(t *Tracer) Query { return t.Trace(all(), e) }
func (e TrueExpr) String() string          { return "TRUE" }

// A > 10 AND A < 5; is always false -> FalseExpr
func isImpossibleRange(min, max any, minInc, maxInc bool) bool {
	switch lo := min.(type) {
	case int64:
		if hi, ok := max.(int64); ok {
			if lo > hi {
				return true
			}
			if lo == hi && (!minInc || !maxInc) {
				return true
			}
		}
	case float64:
		if hi, ok := max.(float64); ok {
			if lo > hi {
				return true
			}
			if lo == hi && (!minInc || !maxInc) {
				return true
			}
		}
	}
	return false
}

func optimize(e Expr) Expr {
	if e == nil {
		return nil
	}

	switch n := e.(type) {
	case BinaryExpr:
		left := optimize(n.Left)
		right := optimize(n.Right)

		if n.Ekind == ExprAnd {
			// AND: with one part FALSE -> result is FALSE
			if _, ok := left.(FalseExpr); ok {
				return FalseExpr{}
			}
			if _, ok := right.(FalseExpr); ok {
				return FalseExpr{}
			}
			// check, that NOT both are FALSE
			if _, ok := left.(TrueExpr); ok {
				return right
			}
			if _, ok := right.(TrueExpr); ok {
				return left
			}
		}
		if n.Ekind == ExprOr {
			// OR: with at least one is TRUE -> result is TRUE
			if _, ok := left.(TrueExpr); ok {
				return TrueExpr{}
			}
			if _, ok := right.(TrueExpr); ok {
				return TrueExpr{}
			}
			// check, that NOT both are TRUE
			if _, ok := left.(FalseExpr); ok {
				return right
			}
			if _, ok := right.(FalseExpr); ok {
				return left
			}
		}
		if n.Ekind == ExprAndNot {
			if _, ok := left.(FalseExpr); ok {
				return FalseExpr{}
			}
			if _, ok := right.(FalseExpr); ok {
				return left
			}
		}

		if n.Ekind == ExprAnd {
			// RULE: And(A, Not(B)) -> AndNot(A, B)
			if notNode, ok := right.(NotExpr); ok {
				return BinaryExpr{Ekind: ExprAndNot, Left: left, Right: notNode.Child}
			}
			// RULE: And(Not(A), B) -> AndNot(A, B)
			if notNode, ok := left.(NotExpr); ok {
				return BinaryExpr{Ekind: ExprAndNot, Left: right, Right: notNode.Child}
			}

			// RULE: And(A > X, B < Y) -> BETWEEN(A, B)
			if lt, okL := left.(TermExpr); okL {
				if rt, okR := right.(TermExpr); okR {
					if lt.Field == rt.Field {
						var min, max any
						var minInc, maxInc bool

						// Identify Lower Bound
						if lt.Op.Op == OpGt || lt.Op.Op == OpGe {
							min, minInc = lt.Value, (lt.Op.Op == OpGe)
						} else if rt.Op.Op == OpGt || rt.Op.Op == OpGe {
							min, minInc = rt.Value, (rt.Op.Op == OpGe)
						}

						// Identify Upper Bound
						if lt.Op.Op == OpLt || lt.Op.Op == OpLe {
							max, maxInc = lt.Value, (lt.Op.Op == OpLe)
						} else if rt.Op.Op == OpLt || rt.Op.Op == OpLe {
							max, maxInc = rt.Value, (rt.Op.Op == OpLe)
						}

						// If we found both a min and a max, we have a BETWEEN
						if min != nil && max != nil {
							if isImpossibleRange(min, max, minInc, maxInc) {
								return FalseExpr{}
							}
							return TermManyExpr{
								Field:  lt.Field,
								Op:     FOpBetween,
								Values: []any{min, max},
							}
						}

					}
				}
			}
		}

		// GC OPTIMIZATION: If nothing was optimized in the children, return the original interface
		// to prevent allocating a new struct on the heap.
		if left.Equals(n.Left) && right.Equals(n.Right) {
			return e
		}

		return BinaryExpr{Ekind: n.Ekind, Left: left, Right: right}

	case NotExpr:
		child := optimize(n.Child)
		switch c := child.(type) {
		// RULE: Not(Not(A)) -> A (Double Negative)
		case NotExpr:
			return optimize(c.Child)

		// DE MORGAN'S LAWS: Push NOT down the tree
		case BinaryExpr:
			if c.Ekind == ExprAnd {
				// RULE: Not(A AND B) -> Not(A) OR Not(B)
				return optimize(BinaryExpr{
					Ekind: ExprOr,
					Left:  NotExpr{Child: c.Left},
					Right: NotExpr{Child: c.Right},
				})
			}
			if c.Ekind == ExprOr {
				// RULE: Not(A OR B) -> Not(A) AND NOT(B)
				return optimize(BinaryExpr{
					Ekind: ExprAnd,
					Left:  NotExpr{Child: c.Left},
					Right: NotExpr{Child: c.Right},
				})
			}

		// RULE: NOT(FALSE) -> TRUE
		case FalseExpr:
			return TrueExpr{}
		// RULE: NOT(TRUE) -> FALSE
		case TrueExpr:
			return FalseExpr{}

		case TermExpr:
			switch c.Op.Op {
			// I'm not sure, that this is faster
			// RULE: NOT (A = B)  -->  A != B
			// case OpEq:
			// 	return NotExpr{Child: TermExpr{Field: c.Field, Op: FOpEq, Value: c.Value}}
			// RULE: NOT (A != B)  -->  A = B
			case OpNeq:
				return TermExpr{Field: c.Field, Op: FOpEq, Value: c.Value}
			// RULE: NOT (A > B) --> A <= B
			case OpGt:
				return TermExpr{Field: c.Field, Op: FOpLe, Value: c.Value}
			// RULE: NOT (A >= B) --> A < B
			case OpGe:
				return TermExpr{Field: c.Field, Op: FOpLt, Value: c.Value}
			// RULE: NOT (A < B) --> A >= B
			case OpLt:
				return TermExpr{Field: c.Field, Op: FOpGe, Value: c.Value}
			// RULE: NOT (A <= B) --> A > B
			case OpLe:
				return TermExpr{Field: c.Field, Op: FOpGt, Value: c.Value}
			}

		}
		// GC OPTIMIZATION: If the child didn't change, return the original wrapper
		if child.Equals(n.Child) {
			return e
		}

		return NotExpr{Child: child}
	}
	return e
}
