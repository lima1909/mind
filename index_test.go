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

	bs, _ := si.Match(FOpEq, "a")
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

	unSet(si, "a", 2)
	bs, _ = si.Match(FOpEq, "a")
	assert.Equal(t, []uint32{1}, bs.ToSlice())

	unSet(si, "a", 1)
	bs, err := si.Match(FOpEq, "a")
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())
}

func TestSortedIndex_Less(t *testing.T) {
	si := NewSortedIndex(FromValue[int]())
	set(si, 1, 1)
	set(si, 1, 2)
	set(si, 3, 3)

	bs, _ := si.Match(FOpLt, 0)
	assert.Equal(t, []uint32{}, bs.ToSlice())
	bs, _ = si.Match(FOpLt, 1)
	assert.Equal(t, []uint32{}, bs.ToSlice())
	bs, _ = si.Match(FOpLt, 2)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(FOpLt, 3)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(FOpLt, 5)
	assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
}

func TestSortedIndex_LessEqual(t *testing.T) {
	si := NewSortedIndex(FromValue[int]())
	set(si, 1, 1)
	set(si, 1, 2)
	set(si, 3, 3)

	bs, _ := si.Match(FOpLe, 0)
	assert.Equal(t, []uint32{}, bs.ToSlice())
	bs, _ = si.Match(FOpLe, 1)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(FOpLe, 2)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
	bs, _ = si.Match(FOpLe, 3)
	assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
	bs, _ = si.Match(FOpLe, 5)
	assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
}

func TestIDIndex_Filter(t *testing.T) {
	mi := newIDMapIndex((*car).Name)
	vw := car{name: "vw", age: 2}
	mi.Set(&vw, 0)

	bs, err := mi.Match(FOpEq, "vw")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{0}, bs.ToSlice())

	_, err = mi.Match(FOpEq, 4)
	assert.ErrorIs(t, InvalidValueTypeError[string]{4}, err)

	_, err = mi.Match(FOpLt, "vw")
	assert.ErrorIs(t, InvalidOperationError{IDMapIndexName, OpLt}, err)

	_, err = mi.Match(FOpEq, "opel")
	assert.ErrorIs(t, ValueNotFoundError{"opel"}, err)
}

func TestSortedIndex_Between_String(t *testing.T) {
	si := NewSortedIndex(FromValue[string]())
	set(si, "a", 1)
	set(si, "a", 2)
	set(si, "b", 3)
	set(si, "c", 4)
	set(si, "x", 5)

	bs, err := si.MatchMany(FOpBetween, "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 4}, bs.ToSlice())

	bs, err = si.MatchMany(FOpBetween, "d", "f")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	bs, err = si.MatchMany(FOpBetween, "x", "z")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{5}, bs.ToSlice())

	bs, err = si.MatchMany(FOpBetween, "a", "a")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

	// from > to
	bs, err = si.MatchMany(FOpBetween, "c", "b")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	// "1" is not in the index
	bs, err = si.MatchMany(FOpBetween, "b", "1")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	// errors
	_, err = si.MatchMany(FOpBetween, "b")
	assert.ErrorIs(t, InvalidArgsLenError{Defined: "2", Got: 1}, err)
}

func TestSortedIndex_Between_Int(t *testing.T) {
	si := NewSortedIndex(FromValue[uint8]())
	set(si, 1, 1)
	set(si, 2, 2)
	set(si, 3, 3)

	// errors
	_, err := si.MatchMany(FOpBetween, "b", "1")
	assert.ErrorIs(t, InvalidValueTypeError[uint8]{"b"}, err)
}

func TestSortedIndex_In_String(t *testing.T) {
	si := NewSortedIndex(FromValue[string]())
	set(si, "a", 1)
	set(si, "a", 2)
	set(si, "b", 3)
	set(si, "c", 4)
	set(si, "x", 5)

	bs, err := si.MatchMany(FOpIn, "b", "c")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 4}, bs.ToSlice())

	bs, err = si.MatchMany(FOpIn, "c", "z")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{4}, bs.ToSlice())

	// not sorted
	bs, err = si.MatchMany(FOpIn, "c", "a")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{1, 2, 4}, bs.ToSlice())

	bs, err = si.MatchMany(FOpIn, "z")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	bs, err = si.MatchMany(FOpIn)
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())

	// empty, because "1" doesn't work
	_, err = si.MatchMany(FOpIn, "b", "1")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{}, bs.ToSlice())
}

func TestSortedIndex_In_Int(t *testing.T) {
	si := NewSortedIndex(FromValue[uint8]())
	set(si, 1, 1)
	set(si, 2, 2)
	set(si, 3, 3)

	// errors
	_, err := si.MatchMany(FOpIn, "b", 1)
	assert.ErrorIs(t, InvalidValueTypeError[uint8]{"b"}, err)
}

func TestIdAutoInc(t *testing.T) {
	obj := struct{}{}

	auto := newIDAutoIncIndex[struct{}]()
	auto.Set(&obj, 1)
	auto.Set(&obj, 2)
	auto.Set(&obj, 5)

	idx, err := auto.GetIndex(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, idx)

	idx, err = auto.GetIndex(2)
	assert.NoError(t, err)
	assert.Equal(t, 2, idx)

	idx, err = auto.GetIndex(3)
	assert.NoError(t, err)
	assert.Equal(t, 5, idx)

	// unset
	auto.UnSet(&obj, 2)
	_, err = auto.GetIndex(2)
	assert.ErrorIs(t, ValueNotFoundError{uint64(2)}, err)

	bs, err := auto.Match(FOpEq, uint64(3))
	assert.NoError(t, err)
	assert.Equal(t, []uint32{5}, bs.ToSlice())
}
