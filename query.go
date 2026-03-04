package main

// Query32 supports only uint32 List-Indices
type Query32 = Query[uint32]

// Query is a filter function, find the correct Index an execute the Index.Get method
// and returns a BitSet pointer
type Query[LI Value] func(l FilterByName[LI], allIDs *BitSet[LI]) (bs *BitSet[LI], canMutate bool, err error)

// FilterByName32 supports only uint32 List-Indices
type FilterByName32 = FilterByName[uint32]

// FilterByName finds the Filter by a given field-name
type FilterByName[LI Value] = func(string) (Filter[LI], error)

// All means returns all Items, no filtering
func All() Query32 { return all[uint32]() }

//go:inline
func all[LI Value]() Query[LI] {
	return func(_ FilterByName[LI], allIDs *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
		return allIDs, false, nil
	}
}

//go:inline
func match[LI Value](fieldName string, op Op, value any) Query[LI] {
	return func(l FilterByName[LI], _ *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Match(op, value)
		return bs, false, err
	}
}

//go:inline
func matchMany[LI Value](fieldName string, op Op, values ...any) Query[LI] {
	return func(l FilterByName[LI], _ *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.MatchMany(op, values...)
		return bs, true, err
	}
}

// ID id = val
func ID(val any) Query32 { return match[uint32](IDIndexFieldName, OpEq, val) }

// Eq fieldName = val
func Eq(fieldName string, val any) Query32 { return match[uint32](fieldName, OpEq, val) }

// Lt Less fieldName < val
func Lt(fieldName string, val any) Query32 { return match[uint32](fieldName, OpLt, val) }

// Le Less Equal fieldName <= val
func Le(fieldName string, val any) Query32 { return match[uint32](fieldName, OpLe, val) }

// Gt Greater fieldName > val
func Gt(fieldName string, val any) Query32 { return match[uint32](fieldName, OpGt, val) }

// Ge Greater Equal fieldName >= val
func Ge(fieldName string, val any) Query32 { return match[uint32](fieldName, OpGe, val) }

// IsNil is a Query which checks for a given type the nil value
func IsNil[V any](fieldName string) Query32 { return isNil[V, uint32](fieldName) }

//go:inline
func isNil[V any, LI Value](fieldName string) Query[LI] {
	return func(l FilterByName[LI], _ *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Match(OpEq, (*V)(nil))
		return bs, false, err
	}
}

// In combines Eq with an Or
// In("name", "Paul", "Egon") => name == "Paul" Or name == "Egon"
func In(fieldName string, vals ...any) Query32 { return in[uint32](fieldName, vals...) }

//go:inline
func in[LI Value](fieldName string, vals ...any) Query[LI] {
	return func(l FilterByName[LI], _ *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
		if len(vals) == 0 {
			return NewEmptyBitSet[LI](), true, nil
		}

		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		bs, err := filter.Match(OpEq, vals[0])
		if err != nil {
			return nil, false, err
		}

		if len(vals) == 1 {
			return bs, false, nil
		}

		bs = bs.Copy()
		for _, val := range vals[1:] {
			bsGet, err := filter.Match(OpEq, val)
			if err != nil {
				return nil, false, err
			}
			bs.Or(bsGet)
		}

		return bs, true, nil
	}
}

// NotEq is a shorcut for Not(Eq(...))
func NotEq(fieldName string, val any) Query32 { return notEq[uint32](fieldName, val) }

//go:inline
func notEq[LI Value](fieldName string, val any) Query[LI] {
	return func(l FilterByName[LI], allIDs *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
		filter, err := l(fieldName)
		if err != nil {
			return nil, false, err
		}

		exclude, err := filter.Match(OpEq, val)
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
func Not[LI Value](q Query[LI]) Query[LI] {
	return func(l FilterByName[LI], allIDs *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
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
func WithPrefix(fieldName string, val string) Query32 {
	return match[uint32](fieldName, OpStartsWith, val)
}

// And combines 2 or more queries with an logical And
func And[LI Value](a Query[LI], b Query[LI], other ...Query[LI]) Query[LI] {
	return func(l FilterByName[LI], allIDs *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
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
func Or[LI Value](a Query[LI], b Query[LI], other ...Query[LI]) Query[LI] {
	return func(l FilterByName[LI], allIDs *BitSet[LI]) (_ *BitSet[LI], canMutate bool, _ error) {
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
func AndNot[LI Value](base Query[LI], sub Query[LI]) Query[LI] {
	return func(l FilterByName[LI], allIDs *BitSet[LI]) (*BitSet[LI], bool, error) {
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
func ensureMutable[LI Value](b *BitSet[LI], canMutate bool, err error) (*BitSet[LI], error) {
	if err != nil {
		return nil, err
	}

	if canMutate {
		return b, nil
	}

	return b.Copy(), nil
}
