package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortedIndex_Equal(t *testing.T) {
	si := NewSortedIndex(FromValue[string]())
	set(si, "a", 1)
	set(si, "a", 2)
	set(si, "b", 3)

	bs, _ := si.Match(OpEq, "a")
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

	unSet(si, "a", 2)
	bs, _ = si.Match(OpEq, "a")
	assert.Equal(t, []uint32{1}, bs.ToSlice())

	unSet(si, "a", 1)
	bs, err := si.Match(OpEq, "a")
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())
}

func TestSortedIndex_Less(t *testing.T) {
	si := NewSortedIndex(FromValue[int]())
	set(si, 1, 1)
	set(si, 1, 2)
	set(si, 3, 3)

	bs, _ := si.Match(OpLt, 0)
	assert.Equal(t, []uint32{}, bs.ToSlice())
	bs, _ = si.Match(OpLt, 1)
	assert.Equal(t, []uint32{}, bs.ToSlice())
	bs, _ = si.Match(OpLt, 2)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(OpLt, 3)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(OpLt, 5)
	assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
}

func TestSortedIndex_LessEqual(t *testing.T) {
	si := NewSortedIndex(FromValue[int]())
	set(si, 1, 1)
	set(si, 1, 2)
	set(si, 3, 3)

	bs, _ := si.Match(OpLe, 0)
	assert.Equal(t, []uint32{}, bs.ToSlice())
	bs, _ = si.Match(OpLe, 1)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(OpLe, 2)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(OpLe, 3)
	assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
	bs, _ = si.Match(OpLe, 5)
	assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
}

func TestIDIndex_Filter(t *testing.T) {
	mi := newIDMapIndex((*car).Name)
	vw := car{name: "vw", age: 2}
	mi.Set(&vw, 0)

	bs, err := mi.Match(OpEq, "vw")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{0}, bs.ToSlice())

	_, err = mi.Match(OpEq, 4)
	assert.ErrorIs(t, InvalidValueTypeError[string]{4}, err)

	_, err = mi.Match(OpLt, "vw")
	assert.ErrorIs(t, InvalidOperationError{IDMapIndexName, OpLt}, err)

	_, err = mi.Match(OpEq, "opel")
	assert.ErrorIs(t, ValueNotFoundError{"opel"}, err)
}

func TestSortedIndex_Between_String(t *testing.T) {
	si := NewSortedIndex(FromValue[string]())
	set(si, "a", 1)
	set(si, "a", 2)
	set(si, "b", 3)
	set(si, "c", 4)
	set(si, "x", 5)

	bs, err := si.MatchMany(OpBetween, "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 4}, bs.ToSlice())

	bs, err = si.MatchMany(OpBetween, "d", "f")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	bs, err = si.MatchMany(OpBetween, "x", "z")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{5}, bs.ToSlice())

	bs, err = si.MatchMany(OpBetween, "a", "a")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

	// from > to
	bs, err = si.MatchMany(OpBetween, "c", "b")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	// "1" is not in the index
	bs, err = si.MatchMany(OpBetween, "b", "1")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	// errors
	_, err = si.MatchMany(OpBetween, "b")
	assert.ErrorIs(t, InvalidArgsLenError{defined: "2", got: 1}, err)
}

func TestSortedIndex_Between_Int(t *testing.T) {
	si := NewSortedIndex(FromValue[uint8]())
	set(si, 1, 1)
	set(si, 2, 2)
	set(si, 3, 3)

	// errors
	_, err := si.MatchMany(OpBetween, "b", "1")
	assert.ErrorIs(t, InvalidValueTypeError[uint8]{"b"}, err)
}

func TestSortedIndex_In_String(t *testing.T) {
	si := NewSortedIndex(FromValue[string]())
	set(si, "a", 1)
	set(si, "a", 2)
	set(si, "b", 3)
	set(si, "c", 4)
	set(si, "x", 5)

	bs, err := si.MatchMany(OpIn, "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 4}, bs.ToSlice())

	bs, err = si.MatchMany(OpIn, "c", "z")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{4}, bs.ToSlice())

	// not sorted
	bs, err = si.MatchMany(OpIn, "c", "a")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{1, 2, 4}, bs.ToSlice())

	bs, err = si.MatchMany(OpIn, "z")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	bs, err = si.MatchMany(OpIn)
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	// empty, because "1" doesn't work
	_, err = si.MatchMany(OpIn, "b", "1")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())
}

func TestSortedIndex_In_Int(t *testing.T) {
	si := NewSortedIndex(FromValue[uint8]())
	set(si, 1, 1)
	set(si, 2, 2)
	set(si, 3, 3)

	// errors
	_, err := si.MatchMany(OpIn, "b", 1)
	assert.ErrorIs(t, InvalidValueTypeError[uint8]{"b"}, err)
}
