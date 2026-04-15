package mind

import (
	"fmt"
	"strings"
	"time"
)

type Expr interface {
	Equals(Expr) bool
	Optimize() Expr
	Compile(*Tracer) Query
	String() string
}

type OrExpr struct {
	Left  Expr
	Right Expr
}

func (e OrExpr) Equals(other Expr) bool {
	o, ok := other.(OrExpr)
	if !ok {
		return false
	}

	return e.Left.Equals(o.Left) && e.Right.Equals(o.Right)
}

func (e OrExpr) Optimize() Expr {

	left := e.Left.Optimize()
	right := e.Right.Optimize()

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

	// GC OPTIMIZATION: If nothing was optimized in the children, return the original interface
	// to prevent allocating a new struct on the heap.
	if left.Equals(e.Left) && right.Equals(e.Right) {
		return e
	}

	return OrExpr{Left: left, Right: right}
}

func (e OrExpr) Compile(t *Tracer) Query {
	var leftTracer, rightTracer *Tracer
	if t != nil {
		leftTracer, rightTracer = &Tracer{}, &Tracer{}
	}

	left := e.Left.Compile(leftTracer)
	right := e.Right.Compile(rightTracer)
	return t.Trace(matchOr(left, right), e, leftTracer, rightTracer)
}

func (e OrExpr) String() string { return fmt.Sprintf("%s OR %s", e.Left, e.Right) }

type AndExpr struct {
	Left  Expr
	Right Expr
}

func (e AndExpr) Equals(other Expr) bool {
	o, ok := other.(AndExpr)
	if !ok {
		return false
	}

	return e.Left.Equals(o.Left) && e.Right.Equals(o.Right)
}
func (e AndExpr) Optimize() Expr {
	left := e.Left.Optimize()
	right := e.Right.Optimize()

	// RULE: And(A, Not(B)) -> AndNot(A, B)
	if notNode, ok := right.(NotExpr); ok {
		return AndNotExpr{Left: left, Right: notNode.Child}
	}
	// RULE: And(Not(A), B) -> AndNot(A, B)
	if notNode, ok := left.(NotExpr); ok {
		return AndNotExpr{Left: right, Right: notNode.Child}
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

	// GC OPTIMIZATION: If nothing was optimized in the children, return the original interface
	// to prevent allocating a new struct on the heap.
	if left.Equals(e.Left) && right.Equals(e.Right) {
		return e
	}

	return AndExpr{Left: left, Right: right}
}

func (e AndExpr) Compile(t *Tracer) Query {
	var leftTracer, rightTracer *Tracer
	if t != nil {
		leftTracer, rightTracer = &Tracer{}, &Tracer{}
	}

	left := e.Left.Compile(leftTracer)
	right := e.Right.Compile(rightTracer)
	return t.Trace(matchAnd(left, right), e, leftTracer, rightTracer)
}
func (e AndExpr) String() string { return fmt.Sprintf("%s AND %s", e.Left, e.Right) }

type AndNotExpr struct {
	Left  Expr
	Right Expr
}

func (e AndNotExpr) Equals(other Expr) bool {
	o, ok := other.(AndNotExpr)
	if !ok {
		return false
	}

	return e.Left.Equals(o.Left) && e.Right.Equals(o.Right)
}

func (e AndNotExpr) Optimize() Expr { return e }

func (e AndNotExpr) Compile(t *Tracer) Query {
	var leftTracer, rightTracer *Tracer
	if t != nil {
		leftTracer, rightTracer = &Tracer{}, &Tracer{}
	}

	left := e.Left.Compile(leftTracer)
	right := e.Right.Compile(rightTracer)
	return t.Trace(matchAndNot(left, right), e, leftTracer, rightTracer)
}

func (e AndNotExpr) String() string { return fmt.Sprintf("%s ANDNOT %s", e.Left, e.Right) }

type NotExpr struct{ Child Expr }

func (e NotExpr) Equals(other Expr) bool {
	o, ok := other.(NotExpr)
	if !ok {
		return false
	}

	return e.Child.Equals(o.Child)
}

func (e NotExpr) Optimize() Expr {

	child := e.Child.Optimize()

	switch c := child.(type) {
	// RULE: Not(Not(A)) -> A (Double Negative)
	case NotExpr:
		return c.Child.Optimize()
	// DE MORGAN'S LAWS: Push NOT down the tree
	case OrExpr:
		// RULE: Not(A OR B) -> Not(A) AND NOT(B)
		return AndExpr{Left: NotExpr{Child: c.Left}, Right: NotExpr{Child: c.Right}}.Optimize()
	case AndExpr:
		// RULE: Not(A AND B) -> Not(A) OR Not(B)
		return OrExpr{Left: NotExpr{Child: c.Left}, Right: NotExpr{Child: c.Right}}.Optimize()

	// RULE: NOT(FALSE) -> TRUE
	case FalseExpr:
		return TrueExpr{}
	// RULE: NOT(TRUE) -> FALSE
	case TrueExpr:
		return FalseExpr{}

	case TermExpr:
		switch c.Op.Op {
		// I think, this rule makes no sense in this context
		// RULE: NOT (A = B)  -->  A != B
		// case OpEq:
		// 	return TermExpr{Field: c.Field, Op: FOpNeq, Value: c.Value}
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
	if child.Equals(e.Child) {
		return e
	}

	return NotExpr{Child: child}
}

func (e NotExpr) Compile(t *Tracer) Query {
	var childTracer *Tracer
	if t != nil {
		childTracer = &Tracer{}
	}

	return t.Trace(matchNot(e.Child.Compile(childTracer)), e, childTracer)
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

	return e.Op == o.Op && e.Field == o.Field && e.Value == o.Value
}

func (e TermExpr) Optimize() Expr { return e }

func (e TermExpr) Compile(t *Tracer) Query {
	switch e.Op.Op {
	case OpEq:
		return t.Trace(matchEqual(e.Field, e.Value), e)
	case OpNeq:
		// is much faster than !=
		// and many Index like Map and SkipList doesn't support !=
		return t.Trace(matchNotEq(e.Field, e.Value), e)
	default:
		return t.Trace(matchOne(e.Field, e.Op, e.Value), e)
	}
}
func (e TermExpr) String() string { return fmt.Sprintf("%s %s %v", e.Field, e.Op, e.Value) }

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

	return e.Op == o.Op && e.Field == o.Field
}

func (e TermManyExpr) Compile(t *Tracer) Query {
	return t.Trace(matchMany(e.Field, e.Op, e.Values...), e)
}
func (e TermManyExpr) Optimize() Expr { return e }
func (e TermManyExpr) String() string { return fmt.Sprintf("%s %s %v", e.Field, e.Op, e.Values) }

// FalseExpr represents a condition that is always false.
// like: A > 10 AND A < 5
type FalseExpr struct{}

func (e FalseExpr) Equals(other Expr) bool  { _, ok := other.(FalseExpr); return ok }
func (e FalseExpr) Compile(t *Tracer) Query { return t.Trace(matchEmpty(), e) }
func (e FalseExpr) Optimize() Expr          { return e }
func (e FalseExpr) String() string          { return "FALSE" }

// TrueExpr represents a condition that is always true.
type TrueExpr struct{}

func (e TrueExpr) Equals(other Expr) bool  { _, ok := other.(TrueExpr); return ok }
func (e TrueExpr) Compile(t *Tracer) Query { return t.Trace(matchAll(), e) }
func (e TrueExpr) Optimize() Expr          { return e }
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

// Tracer collect, what Expr are called and
// how long is the execution durations and
// how many matches are found
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

func (t *Tracer) PrettyString() string {
	if t == nil {
		return "<nil trace>"
	}
	return t.prettyString("", true)
}

func (t *Tracer) prettyString(indent string, isLast bool) string {
	if t == nil {
		return "<nil trace>"
	}

	var sb strings.Builder

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
		sb.WriteString(child.prettyString(newIndent, lastChild))
	}

	return sb.String()
}
