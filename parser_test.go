package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

		{query: `id = 42 and role = "admin"`, expected: []uint32{1}},
		{query: `ok = true or price = 0.0`, expected: []uint32{0}},
		{query: `role = "admin" AND price = 9.9`, expected: []uint32{}},
		{query: `role = "admin" OR price = 9.9`, expected: []uint32{1}},
		{query: `not (ok = true or price = 0.0)`, expected: []uint32{1}},

		//  true or (false and true) => true
		{query: `role = "admin" OR ok = true AND price = 1.2`, expected: []uint32{1}},
		// true or (false and false) => true
		{query: `role = "admin" OR ok = true AND price = 0.0`, expected: []uint32{1}},
		// true or (true and true) => true
		{query: `role = "admin" OR (ok = true AND price = 1.2)`, expected: []uint32{1}},
		// false or (true and true) => true
		{query: `role = "user" OR (ok = false AND price = 1.2)`, expected: []uint32{1}},

		{query: `price between(1.2, 3.0)`, expected: []uint32{0, 1}},
		{query: `price between(3.0, 1.2)`, expected: []uint32{}},

		{query: `price in(1.2, 3.0)`, expected: []uint32{0, 1}},
		{query: `price in(3.0, 1.2)`, expected: []uint32{0, 1}},
		{query: `role in("admin")`, expected: []uint32{1}},
		{query: `role in("nix")`, expected: []uint32{}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			query, err := Parse(tt.query)
			assert.NoError(t, err)

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
			query, err := Parse(tt.query)
			assert.NoError(t, err)

			bs, _, err := query(indexMap.FilterByName, indexMap.allIDs)
			assert.NoError(t, err)
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
			expected_op: OpEq,
			err_op:      OpEOF,
		},
		{
			query:       `role ~`,
			expected_op: OpEq,
			err_op:      OpEOF,
		},
		{
			query:       `false`,
			expected_op: OpIdent,
			err_op:      OpBool,
		},
		{
			query:       `role = `,
			expected_op: OpString,
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
			err_op:      OpNumber,
		},
		{
			query:       `not 3 `,
			expected_op: OpIdent,
			err_op:      OpNumber,
		},
		{
			query:       `role between("1", 2) `,
			expected_op: OpString,
			err_op:      OpNumber,
		},
		{
			query:       `role In(1, "2") `,
			expected_op: OpNumber,
			err_op:      OpString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			_, err := Parse(tt.query)
			var parseErr UnexpectedTokenError
			assert.True(t, errors.As(err, &parseErr))
			assert.Equal(t, tt.err_op, parseErr.token.Op)
			assert.Equal(t, tt.expected_op, parseErr.expected)
		})
	}
}
