package mind

// Query is a filter function, find the correct Index an execute the Index.Get method
// and returns a BitSet pointer
type Query func(l FilterByName, allIDs *RawIDs32) (bs *RawIDs32, canMutate bool, err error)

// FilterByName finds the Filter by a given field-name
type FilterByName = func(string) (Filter, error)

// All means returns all Items, no filtering
func All() Query { return all() }

//go:inline
func all() Query {
	return func(_ FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		return allIDs, false, nil
	}
}

//go:inline
func match(fieldName string, op FilterOp, value any) Query {
	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Match(op, value)
		return bs, false, err
	}
}

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
func ID(val any) Query { return match(IDIndexFieldName, FOpEq, val) }

// Eq fieldName = val
func Eq(fieldName string, val any) Query { return match(fieldName, FOpEq, val) }

// Lt Less fieldName < val
func Lt(fieldName string, val any) Query { return match(fieldName, FOpLt, val) }

// Le Less Equal fieldName <= val
func Le(fieldName string, val any) Query { return match(fieldName, FOpLe, val) }

// Gt Greater fieldName > val
func Gt(fieldName string, val any) Query { return match(fieldName, FOpGt, val) }

// Ge Greater Equal fieldName >= val
func Ge(fieldName string, val any) Query { return match(fieldName, FOpGe, val) }

// IsNil is a Query which checks for a given type the nil value
func IsNil[V any](fieldName string) Query { return isNil[V](fieldName) }

//go:inline
func isNil[V any](fieldName string) Query {
	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Match(FOpEq, (*V)(nil))
		return bs, false, err
	}
}

// In combines Eq with an Or
// In("name", "Paul", "Egon") => name == "Paul" Or name == "Egon"
func In(fieldName string, vals ...any) Query { return in(fieldName, vals...) }

//go:inline
func in(fieldName string, vals ...any) Query {
	return func(l FilterByName, _ *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		if len(vals) == 0 {
			return NewRawIDs[uint32](), true, nil
		}

		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		matched := make([]*RawIDs32, 0, len(vals))
		var maxLen int

		for _, v := range vals {
			rid, err := filter.Match(FOpEq, v)
			if err != nil {
				return nil, false, err
			}

			matched = append(matched, rid)
			rcount := rid.Len()
			if rcount > maxLen {
				maxLen = rcount
			}
		}

		switch len(matched) {
		case 0:
			return NewRawIDs[uint32](), true, nil
		case 1:
			return matched[0], false, nil
		}

		result := NewRawIDsWithCapacity[uint32](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, true, nil
	}
}

// NotEq is a shorcut for Not(Eq(...))
func NotEq(fieldName string, val any) Query { return notEq(fieldName, val) }

//go:inline
func notEq(fieldName string, val any) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		exclude, err := filter.Match(FOpEq, val)
		if err != nil {
			return nil, false, err
		}

		// OPTIMIZATION: If nobody has this value, NotEq is just "All"
		if exclude.Count() == 0 {
			return allIDs, false, nil
		}

		result := allIDs.Copy()
		result.AndNot(exclude)
		return result, true, nil
	}
}

// Not Not(Query)
func Not(q Query) Query {
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

// Eq fieldName = val
func WithPrefix(fieldName string, val string) Query {
	return match(fieldName, FOpStartsWith, val)
}

// And combines 2 or more queries with an logical And
func And(a Query, b Query, other ...Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		result, err := ensureMutable(a(l, allIDs))
		if err != nil {
			return nil, false, err
		}
		right, _, err := b(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		result.And(right)
		// others, if there
		for _, o := range other {
			next, _, err := o(l, allIDs)
			if err != nil {
				return nil, false, err
			}
			result.And(next)
		}

		return result, true, nil
	}
}

// Or combines 2 or more queries with an logical Or
func Or(a Query, b Query, other ...Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (_ *RawIDs32, canMutate bool, _ error) {
		result, err := ensureMutable(a(l, allIDs))
		if err != nil {
			return nil, false, err
		}
		right, _, err := b(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		result.Or(right)
		// others, if there
		for _, o := range other {
			next, _, err := o(l, allIDs)
			if err != nil {
				return nil, false, err
			}
			result.Or(next)
		}

		return result, true, nil
	}
}

// AndNot performs: baseQuery AND NOT(subQuery)
// example: status = 'active' AND type != 'guest'
func AndNot(base Query, sub Query) Query {
	return func(l FilterByName, allIDs *RawIDs32) (*RawIDs32, bool, error) {
		// base result (e.g., the 'active')
		result, canMutate, err := base(l, allIDs)
		if err != nil {
			return nil, false, err
		}

		// early return, if result is false (empty), stop immediately
		if result.IsEmpty() {
			return result, canMutate, nil
		}

		// sub result (e.g., the 'guests')
		exclude, _, err := sub(l, allIDs)
		if err != nil {
			return nil, false, err
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
