package mind

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type User struct {
	ID    int64
	name  string
	role  string
	ok    bool
	price float64
}

func (u *User) Name() string   { return u.name }
func (u *User) Role() string   { return u.role }
func (u *User) Ok() bool       { return u.ok }
func (u *User) Price() float64 { return u.price }

func TestParser_Base(t *testing.T) {
	indexMap := newIndexMap(newIDMapIndex(func(u *User) int64 { return u.ID }))
	indexMap.idIndex.Set(&User{ID: 40}, 0)
	indexMap.idIndex.Set(&User{ID: 42}, 1)
	indexMap.index["name"] = NewSortedIndex((*User).Name)
	indexMap.index["name"].Set(&User{name: "Alice"}, 1)
	indexMap.index["role"] = NewSortedIndex((*User).Role)
	indexMap.index["role"].Set(&User{role: "developer"}, 0)
	indexMap.index["role"].Set(&User{role: "admin"}, 1)
	indexMap.index["price"] = NewSortedIndex((*User).Price)
	indexMap.index["price"].Set(&User{price: 3.0}, 0)
	indexMap.index["price"].Set(&User{price: 1.2}, 1)
	indexMap.index["ok"] = NewMapIndex((*User).Ok)
	indexMap.index["ok"].Set(&User{ok: true}, 0)
	indexMap.index["ok"].Set(&User{ok: false}, 1)
	indexMap.allIDs.Set(0)
	indexMap.allIDs.Set(1)

	tests := []struct {
		query    string
		expected []uint32
	}{
		{query: `id = 42`, expected: []uint32{1}},
		{query: `role="admin"`, expected: []uint32{1}},
		{query: `price = 1.2`, expected: []uint32{1}},
		{query: `price = 4.2`, expected: []uint32{}},
		{query: `ok = false`, expected: []uint32{1}},
		{query: `ok = true`, expected: []uint32{0}},
		{query: `NOT(ok = true)`, expected: []uint32{1}},
		{query: `price < 3.0`, expected: []uint32{1}},
		{query: `price <= 3.0`, expected: []uint32{0, 1}},
		{query: `price > 1.2`, expected: []uint32{0}},
		{query: `price >= 1.2`, expected: []uint32{0, 1}},

		{query: `NOT(role = "admin")`, expected: []uint32{0}},
		// RULE: Not(Not(A)) -> A (Double Negative)
		{query: `NOT(NOT(role = "admin"))`, expected: []uint32{1}},
		// RULE: NOT (A != B)  -->  A = B
		{query: `NOT(role != "admin")`, expected: []uint32{1}},
		// RULE: NOT (A > B) --> A <= B
		{query: `Not(price > 1.2)`, expected: []uint32{1}},
		// RULE: NOT (A >= B) --> A < B
		{query: `Not(price >= 1.3)`, expected: []uint32{1}},
		// RULE: NOT (A < B) --> A >= B
		{query: `Not(price < 3.0)`, expected: []uint32{0}},
		// RULE: NOT (A <= B) --> A > B
		{query: `Not(price <= 2.2)`, expected: []uint32{0}},

		// DE MORGAN'S LAWS: Push NOT down the tree
		// RULE: Not(A AND B) -> Not(A) OR Not(B)
		{query: `Not(price = 1.2 AND price = 3.0)`, expected: []uint32{0, 1}},
		// RULE: Not(A OR B) -> Not(A) AND NOT(B)
		{query: `Not(price = 1.2 OR price = 3.0)`, expected: []uint32{}},

		{query: `id = 42 and role = "admin"`, expected: []uint32{1}},
		{query: `ok = true or price = 0.0`, expected: []uint32{0}},
		{query: `role = "admin" AND price = 9.9`, expected: []uint32{}},
		{query: `role = "admin" OR price = 9.9`, expected: []uint32{1}},
		{query: `not (ok = true or price = 0.0)`, expected: []uint32{1}},

		//  true or (false and true) => true
		{query: `role = "admin" OR ok = tRue AND price = 1.2`, expected: []uint32{1}},
		// true or (false and false) => true
		{query: `role = "admin" OR ok = trUe AND price = 0.0`, expected: []uint32{1}},
		// true or (true and true) => true
		{query: `role = "admin" OR (ok = truE AND price = 1.2)`, expected: []uint32{1}},
		// false or (true and true) => true
		{query: `role = "user" OR (ok = false AND price = 1.2)`, expected: []uint32{1}},

		{query: `price between(1.2, 3.0)`, expected: []uint32{0, 1}},
		{query: `price between(3.0, 1.2)`, expected: []uint32{}},

		{query: `price in(1.2, 3.0)`, expected: []uint32{0, 1}},
		{query: `price in(3.0, 1.2)`, expected: []uint32{0, 1}},
		{query: `role in("developer", "admin")`, expected: []uint32{0, 1}},
		{query: `role in("admin")`, expected: []uint32{1}},
		{query: `role in("developer")`, expected: []uint32{0}},
		{query: `role in("nix")`, expected: []uint32{}},

		{query: `price > 10 and price < 5`, expected: []uint32{}},
		{query: `NOT(price > 10 and price < 5)`, expected: []uint32{0, 1}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			ast, err := Parse(tt.query)
			assert.NoError(t, err)
			ast = ast.Optimize()
			query := ast.Compile(nil)

			bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, bs.ToSlice())
		})
	}

}

func TestParser_Cast(t *testing.T) {
	type data struct {
		U   uint
		U8  uint8
		U16 uint32
		U32 uint32
		U64 uint64

		I   int
		I8  int8
		I16 int16
		I32 int32
		I64 int64

		F32 float32
		F64 float64
	}

	indexMap := newIndexMap[data, struct{}](nil)
	indexMap.index["u"] = NewSortedIndex(FromName[data, uint]("U"))
	indexMap.index["u"].Set(&data{U: 42}, 1)
	indexMap.index["u8"] = NewSortedIndex(FromName[data, uint8]("U8"))
	indexMap.index["u8"].Set(&data{U8: 5}, 1)
	indexMap.index["u16"] = NewSortedIndex(FromName[data, uint16]("U16"))
	indexMap.index["u16"].Set(&data{U16: 16}, 1)
	indexMap.index["u32"] = NewSortedIndex(FromName[data, uint32]("U32"))
	indexMap.index["u32"].Set(&data{U32: 32}, 1)
	indexMap.index["u64"] = NewSortedIndex(FromName[data, uint64]("U64"))
	indexMap.index["u64"].Set(&data{U64: 64}, 1)

	indexMap.index["i"] = NewSortedIndex(FromName[data, int]("I"))
	indexMap.index["i"].Set(&data{I: -42}, 1)
	indexMap.index["i8"] = NewSortedIndex(FromName[data, int8]("I8"))
	indexMap.index["i8"].Set(&data{I8: -8}, 1)
	indexMap.index["i16"] = NewSortedIndex(FromName[data, int16]("I16"))
	indexMap.index["i16"].Set(&data{I16: -16}, 1)
	indexMap.index["i32"] = NewSortedIndex(FromName[data, int32]("I32"))
	indexMap.index["i32"].Set(&data{I32: -32}, 1)
	indexMap.index["i64"] = NewSortedIndex(FromName[data, int64]("I64"))
	indexMap.index["i64"].Set(&data{I64: -64}, 1)

	indexMap.index["f32"] = NewSortedIndex(FromName[data, float32]("F32"))
	indexMap.index["f32"].Set(&data{F32: -3.2}, 1)
	indexMap.index["f64"] = NewSortedIndex(FromName[data, float64]("F64"))
	indexMap.index["f64"].Set(&data{F64: -6.4}, 1)

	tests := []struct {
		query    string
		expected []uint32
	}{
		{query: `u   = 42`, expected: []uint32{1}},
		{query: `u8  = 5`, expected: []uint32{1}},
		{query: `u16 = 16`, expected: []uint32{1}},
		{query: `u32 = 32`, expected: []uint32{1}},
		{query: `u64 = 64`, expected: []uint32{1}},

		{query: `i   = -42`, expected: []uint32{1}},
		{query: `i8  = -8`, expected: []uint32{1}},
		{query: `i16 = -16`, expected: []uint32{1}},
		{query: `i32 = -32`, expected: []uint32{1}},
		{query: `i64 = -64`, expected: []uint32{1}},

		{query: `f32 = -3.2`, expected: []uint32{1}},
		{query: `f64 = -6.4`, expected: []uint32{1}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			ast, err := Parse(tt.query)
			require.NoError(t, err)
			ast = ast.Optimize()
			query := ast.Compile(nil)

			bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, bs.ToSlice())
		})
	}
}

func TestParser_Error(t *testing.T) {

	tests := []struct {
		query       string
		expected_op Op
		err_op      Op
	}{
		{
			query:       ``,
			expected_op: OpIdent,
			err_op:      OpEOF,
		},
		{
			query:       `role`,
			expected_op: OpUndefined,
			err_op:      OpEOF,
		},
		{
			query:       `role ~`,
			expected_op: OpUndefined,
			err_op:      OpEOF,
		},
		{
			query:       `false`,
			expected_op: OpIdent,
			err_op:      OpBool,
		},
		{
			query:       `role = `,
			expected_op: OpUndefined,
			err_op:      OpEOF,
		},
		{
			query:       `(role = 3`,
			expected_op: OpRParen,
			err_op:      OpEOF,
		},
		{
			query:       `role = 3   and `,
			expected_op: OpIdent,
			err_op:      OpEOF,
		},
		{
			query:       `role = 3   and 5 `,
			expected_op: OpIdent,
			err_op:      OpNumberInt,
		},
		{
			query:       `not 3 `,
			expected_op: OpIdent,
			err_op:      OpNumberInt,
		},
		{
			query:       `role between("1", 2) `,
			expected_op: OpString,
			err_op:      OpNumberInt,
		},
		{
			query:       `role In(1, "2") `,
			expected_op: OpNumberInt,
			err_op:      OpString,
		},
		{
			query:       `role = - `,
			expected_op: OpUndefined, // the first expected value
			err_op:      OpUndefined, // unexpected end
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			_, err := Parse(tt.query)
			var parseErr UnexpectedTokenError
			assert.True(t, errors.As(err, &parseErr))
			assert.Equal(t, tt.err_op, parseErr.token.Op)
			assert.Equal(
				t,
				tt.expected_op, parseErr.expected,
				fmt.Sprintf("%q != %q", tt.expected_op, parseErr.expected),
			)
		})
	}
}

func TestParser_Optimize(t *testing.T) {

	tests := []struct {
		input     string
		optimized string
	}{
		{input: `price = 1`, optimized: `price = 1`},
		// RULE: And(A, Not(B)) -> AndNot(A, B)
		{input: `price = 1 and not(price = 2)`, optimized: `price = 1 ANDNOT price = 2`},
		// RULE: And(Not(A), B) -> AndNot(A, B)
		{input: `not(price = 1) and price = 2`, optimized: `price = 2 ANDNOT price = 1`},
		// RULE: And(A > X, B < Y) -> BETWEEN(A, B)
		{input: `price > 1 and price < 2`, optimized: `price BETWEEN [1 2]`},
		// RULE: Not(Not(A)) -> A (Double Negative)
		{input: `not(not(price = 1))`, optimized: `price = 1`},
		// RULE: Not(A AND B) -> Not(A) OR Not(B)
		{input: `not(price = 1 and price = 2)`, optimized: ` NOT(price = 1)  OR  NOT(price = 2) `},
		// RULE: Not(A OR B) -> Not(A) AND NOT(B)
		{input: `not(price = 1 or price = 2)`, optimized: ` NOT(price = 1)  ANDNOT price = 2`},
		// RULE: NOT (A = B)  -->  A != B
		{input: `not(price = 1)`, optimized: ` NOT(price = 1) `},
		// RULE: NOT (A > B) --> A <= B
		{input: `not(price > 1)`, optimized: `price <= 1`},
		// RULE: NOT (A >= B) --> A < B
		{input: `not(price >= 1)`, optimized: `price < 1`},
		// RULE: NOT (A < B) --> A >= B
		{input: `not(price < 1)`, optimized: `price >= 1`},
		// RULE: NOT (A <= B) --> A > B
		{input: `not(price <= 1)`, optimized: `price > 1`},

		// Impossible range: A > 10 AND A < 5 => FALSE
		{input: `price > 10 and price < 5`, optimized: `FALSE`},
		// Impossible range: A >= 10 AND A < 10 => FALSE
		{input: `price >= 10 and price < 10`, optimized: `FALSE`},
		// Impossible range: A > 10 AND A <= 10 => FALSE
		{input: `price > 10 and price <= 10`, optimized: `FALSE`},
		// Equal bounds, both inclusive: A >= 5 AND A <= 5 => valid BETWEEN (not false)
		{input: `price >= 5 and price <= 5`, optimized: `price BETWEEN [5 5]`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser{input: tt.input, lex: lexer{input: tt.input, pos: 0}}
			ast, err := p.parse()
			require.NoError(t, err)
			optimized := ast.Optimize()

			assert.Equal(t, tt.optimized, optimized.String())
		})
	}
}

func TestParser_ConstantFolding(t *testing.T) {
	tests := []struct {
		input     string
		optimized string
	}{
		// FALSE propagation through AND
		// impossible range on left, valid term on right => FALSE
		{input: `(price > 10 and price < 5) and role = 1`, optimized: `FALSE`},
		// valid term on left, impossible range on right => FALSE
		{input: `role = 1 and (price > 10 and price < 5)`, optimized: `FALSE`},

		// FALSE propagation through OR
		// impossible range OR valid term => valid term survives
		{input: `(price > 10 and price < 5) or role = 1`, optimized: `role = 1`},
		// valid term OR impossible range => valid term survives
		{input: `role = 1 or (price > 10 and price < 5)`, optimized: `role = 1`},

		// NOT(FALSE) => TRUE
		{input: `not(price > 10 and price < 5)`, optimized: `TRUE`},

		// TRUE propagation through OR => TRUE
		{input: `not(price > 10 and price < 5) or role = 1`, optimized: `TRUE`},
		{input: `role = 1 or not(price > 10 and price < 5)`, optimized: `TRUE`},

		// TRUE propagation through AND => the other side
		{input: `not(price > 10 and price < 5) and role = 1`, optimized: `role = 1`},
		{input: `role = 1 and not(price > 10 and price < 5)`, optimized: `role = 1`},

		// NOT(TRUE) => FALSE
		{input: `not(not(price > 10 and price < 5))`, optimized: `FALSE`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := parser{input: tt.input, lex: lexer{input: tt.input, pos: 0}}
			ast, err := p.parse()
			require.NoError(t, err)
			optimized := ast.Optimize()

			assert.Equal(t, tt.optimized, optimized.String())
		})
	}
}

func TestParser_ImpossibleRange(t *testing.T) {
	indexMap := newIndexMap(newIDMapIndex(func(u *User) int64 { return u.ID }))
	indexMap.idIndex.Set(&User{ID: 40}, 0)
	indexMap.idIndex.Set(&User{ID: 42}, 1)
	indexMap.index["role"] = NewSortedIndex((*User).Role)
	indexMap.index["role"].Set(&User{role: "developer"}, 0)
	indexMap.index["role"].Set(&User{role: "admin"}, 1)
	indexMap.index["price"] = NewSortedIndex((*User).Price)
	indexMap.index["price"].Set(&User{price: 3.0}, 0)
	indexMap.index["price"].Set(&User{price: 1.2}, 1)
	indexMap.allIDs.Set(0)
	indexMap.allIDs.Set(1)

	tests := []struct {
		query    string
		expected []uint32
	}{
		// Impossible range: price > 10 AND price < 5 => empty
		{query: `price > 10 and price < 5`, expected: []uint32{}},
		// Impossible range ORed with valid condition => valid condition
		{query: `(price > 10 and price < 5) or role = "admin"`, expected: []uint32{1}},
		// Impossible range ANDed with valid condition => empty
		{query: `(price > 10 and price < 5) and role = "admin"`, expected: []uint32{}},
		// NOT(impossible) => all items
		{query: `not(price > 10 and price < 5)`, expected: []uint32{0, 1}},
		// NOT(impossible) AND valid => valid
		{query: `not(price > 10 and price < 5) and role = "admin"`, expected: []uint32{1}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			ast, err := Parse(tt.query)
			assert.NoError(t, err)
			ast = ast.Optimize()
			query := ast.Compile(nil)

			bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, bs.ToSlice())
		})
	}
}

func TestParser_UDF(t *testing.T) {
	indexMap := newIndexMap(newIDMapIndex(func(u *User) int64 { return u.ID }))
	indexMap.idIndex.Set(&User{ID: 40}, 0)
	indexMap.idIndex.Set(&User{ID: 42}, 1)
	indexMap.index["name"] = newUdfIndex((*User).Name)
	indexMap.index["name"].Set(&User{name: "Alice"}, 1)
	indexMap.index["price"] = newUdfIndex((*User).Price)
	indexMap.index["price"].Set(&User{price: 3.0}, 0)
	indexMap.index["price"].Set(&User{price: 1.2}, 1)
	indexMap.allIDs.Set(0)
	indexMap.allIDs.Set(1)

	tests := []struct {
		query    string
		expected []uint32
	}{
		{query: `price my_eq 1.2`, expected: []uint32{1}},
		{query: `price my_eq 4.2`, expected: []uint32{}},
		{query: `Not(price my_eq 4.2)`, expected: []uint32{0, 1}},

		{query: `name my_eq "Alice" AND price = 1.2`, expected: []uint32{1}},
		{query: `name my_eq "Nix" OR price my_eq 3.0`, expected: []uint32{0}},

		{query: `price my_eq(1.2, 3.0)`, expected: []uint32{0, 1}},
		{query: `price my_eq(3.0, 1.2)`, expected: []uint32{0, 1}},

		// without UDF works too
		{query: `price in(1.2, 3.0)`, expected: []uint32{0, 1}},
		{query: `price = 1.2`, expected: []uint32{1}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			ast, err := Parse(tt.query)
			assert.NoError(t, err)
			ast = ast.Optimize()
			query := ast.Compile(nil)

			bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, bs.ToSlice())
		})
	}
}

var udfOp = FilterOp{Op: -1, Name: "my_eq"}

type udfIndex[OBJ any, V comparable, LI UInt] struct {
	data       map[any]*RawIDs[LI]
	fieldGetFn FromField[OBJ, V]
}

func newUdfIndex[OBJ any, V comparable](fromField FromField[OBJ, V]) Index[OBJ] {
	return &udfIndex[OBJ, V, uint32]{
		data:       make(map[any]*RawIDs32),
		fieldGetFn: fromField,
	}
}

func (mi *udfIndex[OBJ, V, LI]) Set(obj *OBJ, lidx LI) {
	value := mi.fieldGetFn(obj)
	bs, found := mi.data[value]
	if !found {
		bs = NewRawIDs[LI]()
	}
	bs.Set(lidx)
	mi.data[value] = bs
}

func (mi *udfIndex[OBJ, V, LI]) UnSet(obj *OBJ, lidx LI) {
	value := mi.fieldGetFn(obj)
	if bs, found := mi.data[value]; found {
		bs.UnSet(lidx)
		if bs.Count() == 0 {
			delete(mi.data, value)
		}
	}
}

func (mi *udfIndex[OBJ, V, LI]) HasChanged(oldItem, newItem *OBJ) bool {
	return mi.fieldGetFn(oldItem) != mi.fieldGetFn(newItem)
}

func (mi *udfIndex[OBJ, V, LI]) Equal(value any) (*RawIDs[LI], error) {
	v, err := ValueFromAny[V](value)
	if err != nil {
		return nil, InvalidValueTypeError[V]{value}
	}

	bs, found := mi.data[v]
	if !found {
		return NewRawIDs[LI](), nil
	}
	return bs, nil
}

func (mi *udfIndex[OBJ, V, LI]) Match(op FilterOp, value any) (*RawIDs[LI], error) {
	if op != udfOp {
		return nil, InvalidOperationError{MapIndexName, op.Op}
	}

	return mi.Equal(value)
}

// MatchMany is not supported by MapIndex, so that always returns an error
func (mi *udfIndex[OBJ, V, LI]) MatchMany(op FilterOp, values ...any) (*RawIDs[LI], error) {
	switch op {
	case udfOp, FOpIn:
		if len(values) == 0 {
			return NewRawIDs[LI](), nil
		}

		matched := make([]*RawIDs[LI], 0, len(values))
		var maxLen int

		for _, v := range values {
			key, err := ValueFromAny[V](v)
			if err != nil {
				return nil, err
			}

			if rid, found := mi.data[key]; found {
				matched = append(matched, rid)
				rcount := rid.Count()
				if rcount > maxLen {
					maxLen = rcount
				}
			}
		}

		if len(matched) == 0 {
			return NewRawIDs[LI](), nil
		}

		result := NewRawIDsWithCapacity[LI](maxLen)
		for _, bs := range matched {
			result.Or(bs)
		}

		return result, nil
	default:
		return nil, InvalidOperationError{MapIndexName, op.Op}
	}
}
