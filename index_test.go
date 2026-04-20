package mind

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndex_EquaString(t *testing.T) {
	index := []struct {
		name  string
		index Index[string]
	}{
		{"map", NewMapIndex(FromValue[string]())},
		{"sorted", NewSortedIndex(FromValue[string]())},
	}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, "a", 1)
			set(tt.index, "a", 2)
			set(tt.index, "b", 3)

			bs, _ := tt.index.Equal("a")
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

			unSet(tt.index, "a", 2)
			bs, _ = tt.index.Equal("a")
			assert.Equal(t, []uint32{1}, bs.ToSlice())

			unSet(tt.index, "a", 1)
			bs, err := tt.index.Equal("a")
			assert.NoError(t, err)
			assert.Equal(t, 0, bs.Count())
		})
	}

}

func TestRangeIndex_Delete(t *testing.T) {
	i := NewRangeIndex(FromValue[uint8]())
	set(i, 1, 1)
	set(i, 1, 2)
	set(i, 2, 2)
	set(i, 9, 9)

	ri := i.(*RangeIndex[uint8])
	assert.Equal(t, 10, ri.max)

	var del uint8 = 9
	i.UnSet(&del, 9)
	assert.Equal(t, 3, ri.max)

	del = 7
	i.UnSet(&del, 9)
	assert.Equal(t, 3, ri.max)

	del = 2
	i.UnSet(&del, 2)
	assert.Equal(t, 2, ri.max)

	del = 1
	i.UnSet(&del, 2)
	assert.Equal(t, 2, ri.max)
	del = 1
	i.UnSet(&del, 1)
	assert.Equal(t, 0, ri.max)

	// max value and greater int index
	set(i, 255, 2560)
	assert.Equal(t, 256, ri.max)

	set(i, 0, 1)
	assert.Equal(t, 256, ri.max)

	del = 255
	i.UnSet(&del, 2560)
	assert.Equal(t, 1, ri.max)
}

type testIndex struct {
	name  string
	index Index[uint8]
}

func index() []testIndex {
	return []testIndex{
		{"sorted", NewSortedIndex(FromValue[uint8]())},
		{"range", NewRangeIndex(FromValue[uint8]())},
	}
}

func TestIndex_Empty(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			bs, err := tt.index.Equal(1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, err = tt.index.Match(FOpLt, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, err = tt.index.Match(FOpLe, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, err = tt.index.Match(FOpGt, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, err = tt.index.Match(FOpGe, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIndex_Equal(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			bs, err := tt.index.Equal(0)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, err = tt.index.Equal(1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

			bs, err = tt.index.Equal(5)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIndex_Less(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			bs, _ := tt.index.Match(FOpLt, 0)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLt, 1)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLt, 2)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLt, 3)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLt, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
		})
	}
}

func TestIndex_LessEqual(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			bs, _ := tt.index.Match(FOpLe, 0)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLe, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLe, 2)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLe, 3)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpLe, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
		})
	}
}

func TestIndex_Greater(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			bs, _ := tt.index.Match(FOpGt, 0)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGt, 1)
			assert.Equal(t, []uint32{3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGt, 2)
			assert.Equal(t, []uint32{3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGt, 3)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGt, 5)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIndex_GreaterEqual(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			bs, _ := tt.index.Match(FOpGe, 0)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGe, 1)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGe, 2)
			assert.Equal(t, []uint32{3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGe, 3)
			assert.Equal(t, []uint32{3}, bs.ToSlice())
			bs, _ = tt.index.Match(FOpGe, 5)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIndex_Between(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)
			set(tt.index, 255, 255)

			bs, _ := tt.index.MatchMany(FOpBetween, 0, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpBetween, 1, 2)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpBetween, 1, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			bs, _ = tt.index.MatchMany(FOpBetween, 1, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpBetween, 1, 3)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpBetween, 0, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			bs, _ = tt.index.MatchMany(FOpBetween, 0, 255)
			assert.Equal(t, []uint32{1, 2, 3, 255}, bs.ToSlice())

			bs, _ = tt.index.MatchMany(FOpBetween, 2, 1)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			_, err := tt.index.MatchMany(FOpBetween, 1, 256)
			assert.ErrorIs(t, InvalidValueTypeError[uint8]{256}, err)
		})
	}
}

func TestIndex_In(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			bs, _ := tt.index.MatchMany(FOpIn, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpIn, 0, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpIn, 3, 0, 1)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _ = tt.index.MatchMany(FOpIn, 5, 3, 0, 1)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			bs, _ = tt.index.MatchMany(FOpIn, 0, 2, 5)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIDIndex_Filter(t *testing.T) {
	mi := newIDMapIndex((*car).Name)
	vw := car{name: "vw", age: 2}
	mi.Set(&vw, 0)

	bs, err := mi.Equal("vw")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{0}, bs.ToSlice())

	_, err = mi.Equal(4)
	assert.ErrorIs(t, InvalidValueTypeError[string]{4}, err)

	_, err = mi.Match(FOpLt, "vw")
	assert.ErrorIs(t, InvalidOperationError{IDMapIndexName, OpLt}, err)

	_, err = mi.Equal("opel")
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

func TestSortedIndex_Between_Error(t *testing.T) {
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

func TestIndex_BulkSet(t *testing.T) {

	index := []struct {
		name  string
		index Index[uint8]
	}{
		{"map", NewMapIndex(FromValue[uint8]())},
		{"sorted", NewSortedIndex(FromValue[uint8]())},
		{"range", NewRangeIndex(FromValue[uint8]())},
		{"idMap", newIDMapIndex(FromValue[uint8]())},
	}

	zero := uint8(0)
	two := uint8(2)
	eigth := uint8(8)
	values := []*uint8{&zero, &two, &eigth}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			tt.index.BulkSet(slices.All(values))

			bs, err := tt.index.Equal(zero)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{0}, bs.ToSlice())

			bs, err = tt.index.Equal(eigth)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{2}, bs.ToSlice())
		})
	}
}
