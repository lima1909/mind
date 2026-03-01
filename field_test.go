package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestField_ValueFromString_(t *testing.T) {
	// String
	s, err := ValueFromString[string]("foo")
	assert.NoError(t, err)
	assert.Equal(t, "foo", s)

	sPtr, err := ValueFromString[*string]("foo")
	assert.NoError(t, err)
	foo := "foo"
	assert.Equal(t, &foo, sPtr)

	// Boolean
	b, err := ValueFromString[bool]("True")
	assert.NoError(t, err)
	assert.Equal(t, true, b)
	b, err = ValueFromString[bool]("False")
	assert.NoError(t, err)
	assert.Equal(t, false, b)
	bPtr, err := ValueFromString[*bool]("false")
	assert.NoError(t, err)
	assert.Equal(t, new(bool), bPtr)

	// Number
	i, err := ValueFromString[int]("-42")
	assert.NoError(t, err)
	assert.Equal(t, -42, i)

	u, err := ValueFromString[uint]("42")
	assert.NoError(t, err)
	assert.Equal(t, uint(42), u)

	u8, err := ValueFromString[uint8]("42")
	assert.NoError(t, err)
	assert.Equal(t, uint8(42), u8)

	f32, err := ValueFromString[float32]("-4.2")
	assert.NoError(t, err)
	assert.Equal(t, float32(-4.2), f32)

	f64, err := ValueFromString[float64]("-4.2")
	assert.NoError(t, err)
	assert.Equal(t, -4.2, f64)
}

func TestField_ValueFromString_Error_(t *testing.T) {
	// to big error
	_, err := ValueFromString[uint8]("420")
	assert.Error(t, err)
}

func TestField_FieldValues(t *testing.T) {
	fv := NewFieldValues[int]("1", "-42")

	first, ok, err := fv.First()
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, first)

	v, ok, err := fv.Next()
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	v, ok, err = fv.Next()
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, -42, v)

	_, ok, err = fv.Next()
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestField_FieldValues_Errors(t *testing.T) {
	fv := NewFieldValues[int]("")

	_, ok, err := fv.First()
	assert.False(t, ok)
	// empty string error
	assert.Error(t, err)

	_, ok, err = fv.Next()
	// empty string error
	assert.Error(t, err)
	assert.False(t, ok)

	fv = NewFieldValues[int]()

	_, ok, err = fv.First()
	assert.False(t, ok)
	assert.NoError(t, err)

	_, ok, err = fv.Next()
	assert.False(t, ok)
	assert.NoError(t, err)
}
