package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

//	func TestField_ValueFromString_(t *testing.T) {
//		// String
//		sPtr, err := ValueFromString[*string]("foo")
//		assert.NoError(t, err)
//		foo := "foo"
//		assert.Equal(t, &foo, sPtr)
//
//		// Boolean
//		b, err := ValueFromString[bool]("True")
//		assert.NoError(t, err)
//		assert.Equal(t, true, b)
//		b, err = ValueFromString[bool]("False")
//		assert.NoError(t, err)
//		assert.Equal(t, false, b)
//		bPtr, err := ValueFromString[*bool]("false")
//		assert.NoError(t, err)
//		assert.Equal(t, new(bool), bPtr)
//
//		// Number
//		i, err := ValueFromString[int]("-42")
//		assert.NoError(t, err)
//		assert.Equal(t, -42, i)
//
//		u, err := ValueFromString[uint]("42")
//		assert.NoError(t, err)
//		assert.Equal(t, uint(42), u)
//
//		u8, err := ValueFromString[uint8]("42")
//		assert.NoError(t, err)
//		assert.Equal(t, uint8(42), u8)
//
//		f32, err := ValueFromString[float32]("-4.2")
//		assert.NoError(t, err)
//		assert.Equal(t, float32(-4.2), f32)
//
//		f64, err := ValueFromString[float64]("-4.2")
//		assert.NoError(t, err)
//		assert.Equal(t, -4.2, f64)
//	}
//
//	func TestField_ValueFromString_Error_(t *testing.T) {
//		// to big error
//		_, err := ValueFromString[uint8]("420")
//		assert.Error(t, err)
//	}

func TestField_ValueFromAny_Int64(t *testing.T) {
	// int64 -> int
	value, err := ValueFromAny[int](int64(-123456))
	assert.NoError(t, err)
	assert.Equal(t, -123456, value)

	// int64 -> int64
	v64, err := ValueFromAny[int64](int64(-123456))
	assert.NoError(t, err)
	assert.Equal(t, int64(-123456), v64)

	// int64 -> int32
	v32, err := ValueFromAny[int32](int64(-123456))
	assert.NoError(t, err)
	assert.Equal(t, int32(-123456), v32)

	// int64 -> int16
	v16, err := ValueFromAny[int16](int64(-123))
	assert.NoError(t, err)
	assert.Equal(t, int16(-123), v16)

	// int64 -> int8
	v8, err := ValueFromAny[int8](int64(-123))
	assert.NoError(t, err)
	assert.Equal(t, int8(-123), v8)

	// int64 -> uint
	u, err := ValueFromAny[uint](int64(123456))
	assert.NoError(t, err)
	assert.Equal(t, uint(123456), u)

	// int64 -> uint64
	u64, err := ValueFromAny[uint64](int64(123456))
	assert.NoError(t, err)
	assert.Equal(t, uint64(123456), u64)

	// int64 -> uint32
	u32, err := ValueFromAny[uint32](int64(123456))
	assert.NoError(t, err)
	assert.Equal(t, uint32(123456), u32)

	// int64 -> uint16
	u16, err := ValueFromAny[uint16](int64(1234))
	assert.NoError(t, err)
	assert.Equal(t, uint16(1234), u16)

	// int64 -> uint8
	u8, err := ValueFromAny[uint8](int64(123))
	assert.NoError(t, err)
	assert.Equal(t, uint8(123), u8)

	// int -> int32
	i32, err := ValueFromAny[int32](-123456)
	assert.NoError(t, err)
	assert.Equal(t, int32(-123456), i32)

	// float64 -> float64
	f64, err := ValueFromAny[float64](float64(-123.456))
	assert.NoError(t, err)
	assert.Equal(t, float64(-123.456), f64)

	// float64 -> float32
	f32, err := ValueFromAny[float32](float64(-123.456))
	assert.NoError(t, err)
	assert.Equal(t, float32(-123.456), f32)

	// string -> string
	s, err := ValueFromAny[string]("hallo")
	assert.NoError(t, err)
	assert.Equal(t, "hallo", s)

	// bool -> bool
	b, err := ValueFromAny[bool](true)
	assert.NoError(t, err)
	assert.Equal(t, true, b)

	// rune -> rune
	r, err := ValueFromAny[rune]('X')
	assert.NoError(t, err)
	assert.Equal(t, 'X', r)
}
