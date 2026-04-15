package mind

// Query is a filter function, find the correct Index an execute the Index.Get method
// and returns a BitSet pointer
type Query func(l FilterByName, allIDs *RawIDs32) (bs *RawIDs32, canMutate bool, err error)

// FilterByName finds the Filter by a given field-name
type FilterByName = func(string) (Filter, error)

// All means returns all Items, no filtering
func All() Expr { return TrueExpr{} }

// all returns always an allIDs
//
//go:inline
func matchAll() Query {
	return func(_ FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		return allIDs, false, nil
	}
}

// empty returns always an empty RawIDs, the opposite of all
//
//go:inline
func matchEmpty() Query {
	return func(_ FilterByName, _ *RawIDs32) (*RawIDs32, bool, error) {
		return NewRawIDs[uint32](), true, nil
	}
}

// matchEqual matched Equal with given value
//
//go:inline
func matchEqual(fieldName string, value any) Query {
	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Equal(value)
		return bs, false, err
	}
}

// matchOne matched ONE given value
//
//go:inline
func matchOne(fieldName string, op FilterOp, value any) Query {
	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Match(op, value)
		return bs, true, err
	}
}

// matchMany matched MANY given value
//
//go:inline
func matchMany(fieldName string, op FilterOp, values ...any) Query {
	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.MatchMany(op, values...)
		return bs, true, err
	}
}

// ID id = val
func ID(val any) Expr { return TermExpr{IDIndexFieldName, FOpEq, val} }

// Eq fieldName = val
func Eq(fieldName string, val any) Expr { return TermExpr{fieldName, FOpEq, val} }

// Lt Less fieldName < val
func Lt(fieldName string, val any) Expr { return TermExpr{fieldName, FOpLt, val} }

// Le Less Equal fieldName <= val
func Le(fieldName string, val any) Expr { return TermExpr{fieldName, FOpLe, val} }

// Gt Greater fieldName > val
func Gt(fieldName string, val any) Expr { return TermExpr{fieldName, FOpGt, val} }

// Ge Greater Equal fieldName >= val
func Ge(fieldName string, val any) Expr { return TermExpr{fieldName, FOpGe, val} }

// IsNil is a Query which checks for a given type the nil value
// func IsNil[V any](fieldName string) Query { return isNil[V](fieldName) }

//go:inline
// func isNil[V any](fieldName string) Query {
// 	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
// 		filter, err := l(fieldName)
// 		if err != nil {
// 			return nil, false, err
// 		}
//
// 		bs, err := filter.Match(FOpEq, (*V)(nil))
// 		return bs, false, err
// 	}
// }

// In combines Eq with an Or
// In("age", 21, 42) => age == 21 Or age == 42
func In(fieldName string, vals ...any) Expr { return TermManyExpr{fieldName, FOpIn, vals} }

// NotEq is a shorcut for Not(Eq(...)) and means for example age != 42
func NotEq(fieldName string, val any) Expr { return TermExpr{fieldName, FOpNeq, val} }

//go:inline
func matchNotEq(fieldName string, val any) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		exclude, err := filter.Equal(val)
		if err != nil {
			return nil, false, err
		}

		// OPTIMIZATION: If nobody has this value, NotEq is just "All"
		if exclude.IsEmpty() {
			return allIDs, false, nil
		}

		result := allIDs.Copy()
		result.AndNot(exclude)
		return result, true, nil
	}
}

// Not Not(Query), for example Not(age > 42)
func Not(expr Expr) Expr { return NotExpr{expr} }

// Not Not(Query)
//
//go:inline
func matchNot(q Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		// can Mutate is not relevant, because allIDs are copied
		qres, _, err := q(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		// maybe i can change the copy?
		result := allIDs.Copy()
		result.AndNot(qres)
		return result, true, nil
	}
}

// WithPrefix query for string starts with
func WithPrefix(fieldName string, val string) Query { return matchOne(fieldName, FOpStartsWith, val) }

// And combines 2 or more queries with an logical And
func And(left Expr, right Expr) Expr { return AndExpr{left, right} }

//go:inline
func matchAnd(a Query, b Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {

		result, canMutate, err := a(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		// if Query 'a' has 0 matches, stop immediately.
		// we completely skip executing 'b' and 'other'.
		if result.IsEmpty() {
			return result, canMutate, nil
		}

		result, err = ensureMutable(result, canMutate, nil)
		if err != nil {
			return nil, false, err
		}

		right, _, err := b(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		result.And(right)
		return result, true, nil
	}
}

// Or combines 2 or more queries with an logical Or
func Or(left Expr, right Expr) Expr { return OrExpr{left, right} }

//go:inline
func matchOr(a Query, b Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		result, canMutate, err := a(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		right, rightMutate, err := b(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		// if 'result' was empty, the result is just 'right'.
		if result.IsEmpty() {
			return right, rightMutate, nil
		}

		if !right.IsEmpty() {
			// both have data, so we must merge them.
			result, err = ensureMutable(result, canMutate, nil)
			if err != nil {
				return nil, false, err
			}
			result.Or(right)
			canMutate = true
		}

		return result, canMutate, nil
	}
}

// AndNot performs: baseQuery AND NOT(subQuery)
// example: status = 'active' AND type != 'guest'
func AndNot(left Expr, right Expr) Expr { return AndNotExpr{left, right} }

//go:inline
func matchAndNot(base Query, sub Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (*RawIDs32, bool, error) {
		// base result (e.g., the 'active')
		result, canMutate, err := base(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		// early return, if result is false (empty), stop immediately
		// 0 - B = 0
		if result.IsEmpty() {
			return result, canMutate, nil
		}

		// sub result (e.g., the 'guests')
		exclude, _, err := sub(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		// if 'b' is empty, A - 0 = A.
		// We can return 'result' exactly as-is without allocating or mutating.
		if exclude.IsEmpty() {
			return result, canMutate, nil
		}

		result, err = ensureMutable(result, canMutate, nil)
		if err != nil {
			return nil, false, err
		}

		result.AndNot(exclude)

		return result, true, nil
	}
}

// check, must the BitSet copied or not
// only copy, if not mutable
//
//go:inline
func ensureMutable(rid *RawIDs32, canMutate bool, err error) (*RawIDs32, error) {
	if err != nil {
		return nil, err
	}

	if canMutate {
		return rid, nil
	}

	return rid.Copy(), nil
}
