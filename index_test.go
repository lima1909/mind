package mind

import (
	"slices"
	"testing"

	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
	"github.com/stretchr/testify/assert"
)

func TestIndex_EqualString(t *testing.T) {
	index := []struct {
		name  string
		index Index[string]
	}{
		{"map", NewMapIndex(FromValue[string]())},
		{"sorted", NewSortedIndex(FromValue[string]())},
		{"string", NewStringIndex(FromValue[string]())},
		{"composite", NewMapCompositeIndex(FromValue[string]())},
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

	ri := i.(*RangeIndex[uint8, SingleValueHandler[uint8, uint8]])
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
		{"rangeencoded", NewCompositeIndex(NewMapIndex(FromValue[uint8]())).
			Add(NewRangeEncodedIndex(FromValue[uint8](), 255), query.FOpLe, query.FOpLt, query.FOpGe, query.FOpGt, query.FOpBetween)},
		{"fenwick", NewCompositeIndex(NewMapIndex(FromValue[uint8]())).
			Add(NewFenwickIndex(FromValue[uint8](), 255), query.FOpLe, query.FOpLt, query.FOpGe, query.FOpGt, query.FOpBetween)},
	}
}

func TestIndex_Empty(t *testing.T) {
	allIDs := lidx.NewRawIDs[uint32]()

	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			bs, err := tt.index.Equal(1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpLt, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpLe, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpGt, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpGe, 1)
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
			set(tt.index, 255, 255)

			allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3, 255)

			bs, canMutate, _ := tt.index.Match(allIDs, query.FOpLt, 0)
			assert.True(t, canMutate)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLt, 1)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLt, 2)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLt, 3)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLt, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLt, 255)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			_, _, err := tt.index.Match(allIDs, query.FOpLt, 256)
			assert.Error(t, err)
		})
	}
}

func TestIndex_LessEqual(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)
			set(tt.index, 255, 255)

			allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3, 255)

			bs, _, _ := tt.index.Match(allIDs, query.FOpLe, 0)
			assert.Equal(t, []uint32{}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLe, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLe, 2)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLe, 3)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLe, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpLe, 255)
			assert.Equal(t, []uint32{1, 2, 3, 255}, bs.ToSlice())

			_, _, err := tt.index.Match(allIDs, query.FOpLe, 256)
			assert.Error(t, err)
		})
	}
}

func TestIndex_Greater(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)
			set(tt.index, 255, 255)

			allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3, 255)

			bs, canMutate, _ := tt.index.Match(allIDs, query.FOpGt, 0)
			assert.True(t, canMutate)
			assert.Equal(t, []uint32{1, 2, 3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGt, 1)
			assert.Equal(t, []uint32{3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGt, 2)
			assert.Equal(t, []uint32{3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGt, 3)
			assert.Equal(t, []uint32{255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGt, 5)
			assert.Equal(t, []uint32{255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGt, 255)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			_, _, err := tt.index.Match(allIDs, query.FOpGt, 256)
			assert.Error(t, err)
		})
	}
}

func TestIndex_GreaterEqual(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)
			set(tt.index, 255, 255)

			allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3, 255)

			bs, _, _ := tt.index.Match(allIDs, query.FOpGe, 0)
			assert.Equal(t, []uint32{1, 2, 3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGe, 1)
			assert.Equal(t, []uint32{1, 2, 3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGe, 2)
			assert.Equal(t, []uint32{3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGe, 3)
			assert.Equal(t, []uint32{3, 255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGe, 5)
			assert.Equal(t, []uint32{255}, bs.ToSlice())
			bs, _, _ = tt.index.Match(allIDs, query.FOpGe, 255)
			assert.Equal(t, []uint32{255}, bs.ToSlice())

			_, _, err := tt.index.Match(allIDs, query.FOpGe, 256)
			assert.Error(t, err)
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

			bs, _, _ := tt.index.MatchMany(query.FOpBetween, 0, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 1, 2)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 1, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 1, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 1, 3)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 0, 5)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 0, 255)
			assert.Equal(t, []uint32{1, 2, 3, 255}, bs.ToSlice())

			bs, _, _ = tt.index.MatchMany(query.FOpBetween, 2, 1)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			_, _, err := tt.index.MatchMany(query.FOpBetween, 1, 256)
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

			bs, _, _ := tt.index.MatchMany(query.FOpIn, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpIn, 0, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpIn, 3, 0, 1)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())
			bs, _, _ = tt.index.MatchMany(query.FOpIn, 5, 3, 0, 1)
			assert.Equal(t, []uint32{1, 2, 3}, bs.ToSlice())

			bs, _, _ = tt.index.MatchMany(query.FOpIn, 0, 2, 5)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIndex_UnSet(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 1, 2)
			set(tt.index, 3, 3)

			allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3)

			bs, _, _ := tt.index.MatchMany(query.FOpIn, 1)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

			// remove
			unSet(tt.index, 1, 2)
			allIDs.UnSet(2)

			bs, _, _ = tt.index.MatchMany(query.FOpIn, 1)
			assert.Equal(t, []uint32{1}, bs.ToSlice())

			bs, _, err := tt.index.Match(allIDs, query.FOpLe, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1}, bs.ToSlice())

			// remove
			unSet(tt.index, 1, 1)
			allIDs.UnSet(1)

			bs, _, _ = tt.index.MatchMany(query.FOpIn, 1)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpLe, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestIndex_List(t *testing.T) {
	for _, tt := range index() {
		t.Run(tt.name, func(t *testing.T) {
			l := NewList[uint8]()
			assert.NoError(t, l.CreateIndex("value", tt.index))

			assert.Equal(t, 0, l.Insert(1))
			assert.Equal(t, 1, l.Insert(1))
			assert.Equal(t, 2, l.Insert(3))

			result, err := l.Query(query.Lt("value", 3)).Values()
			assert.NoError(t, err)
			assert.Equal(t, []uint8{1, 1}, result)

			result, err = l.QueryStr("value between(1, 3)").Values()
			assert.NoError(t, err)
			assert.Equal(t, []uint8{1, 1, 3}, result)
		})
	}
}

func TestIDIndex_Filter(t *testing.T) {
	mi := newIDMapIndex((*car).Name)
	vw := car{name: "vw", age: 2}
	mi.Set(&vw, 0)

	allIDS := lidx.NewRawIDsFrom[uint32](0)

	bs, err := mi.Equal("vw")
	assert.NoError(t, err)
	assert.Equal(t, []uint32{0}, bs.ToSlice())

	_, err = mi.Equal(4)
	assert.ErrorIs(t, InvalidValueTypeError[string]{4}, err)

	_, _, err = mi.Match(allIDS, query.FOpLt, "vw")
	assert.ErrorIs(t, InvalidOperationError{IDMapIndexName, query.OpLt}, err)

	_, err = mi.Equal("opel")
	assert.ErrorIs(t, ValueNotFoundError{"opel"}, err)
}

func TestIndex_Between_String(t *testing.T) {
	index := []struct {
		name  string
		index Index[string]
	}{
		{"sorted", NewSortedIndex(FromValue[string]())},
		{"string", NewStringSortedIndex(FromValue[string]())},
		{"composite", NewCompositeIndex(NewMapIndex(FromValue[string]())).
			Add(NewSortedIndex(FromValue[string]()), query.FOpBetween)},
	}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, "a", 1)
			set(tt.index, "a", 2)
			set(tt.index, "b", 3)
			set(tt.index, "c", 4)
			set(tt.index, "x", 5)

			bs, canMutate, err := tt.index.MatchMany(query.FOpBetween, "b", "c")
			assert.True(t, canMutate)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{3, 4}, bs.ToSlice())

			bs, _, err = tt.index.MatchMany(query.FOpBetween, "d", "f")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.MatchMany(query.FOpBetween, "x", "z")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{5}, bs.ToSlice())

			bs, _, err = tt.index.MatchMany(query.FOpBetween, "a", "a")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1, 2}, bs.ToSlice())

			// from > to
			bs, _, err = tt.index.MatchMany(query.FOpBetween, "c", "b")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			// "1" is not in the index
			bs, _, err = tt.index.MatchMany(query.FOpBetween, "b", "1")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			// errors
			_, _, err = tt.index.MatchMany(query.FOpBetween, "b")
			assert.ErrorIs(t, InvalidArgsLenError{Defined: "2", Got: 1}, err)
		})
	}
}

func TestSortedIndex_Between_Error(t *testing.T) {
	si := NewSortedIndex(FromValue[uint8]())
	set(si, 1, 1)
	set(si, 2, 2)
	set(si, 3, 3)

	// errors
	_, _, err := si.MatchMany(query.FOpBetween, "b", "1")
	assert.ErrorIs(t, InvalidValueTypeError[uint8]{"b"}, err)
}

func TestIndex_In_String(t *testing.T) {
	index := []struct {
		name  string
		index Index[string]
	}{
		{"sorted", NewSortedIndex(FromValue[string]())},
		{"string", NewStringIndex(FromValue[string]())},
		{"composite", NewCompositeIndex(NewMapIndex(FromValue[string]())).
			Add(NewSortedIndex(FromValue[string]()), query.FOpIn)},
	}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, "a", 1)
			set(tt.index, "a", 2)
			set(tt.index, "b", 3)
			set(tt.index, "c", 4)
			set(tt.index, "x", 5)

			bs, _, err := tt.index.MatchMany(query.FOpIn, "b", "c")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{3, 4}, bs.ToSlice())

			bs, _, err = tt.index.MatchMany(query.FOpIn, "c", "z")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{4}, bs.ToSlice())

			// not sorted
			bs, _, err = tt.index.MatchMany(query.FOpIn, "c", "a")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1, 2, 4}, bs.ToSlice())

			bs, _, err = tt.index.MatchMany(query.FOpIn, "z")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			bs, _, err = tt.index.MatchMany(query.FOpIn)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())

			// empty, because "1" doesn't work
			_, _, err = tt.index.MatchMany(query.FOpIn, "b", "1")
			assert.NoError(t, err)
			assert.Equal(t, []uint32{}, bs.ToSlice())
		})
	}
}

func TestSortedIndex_In_Int(t *testing.T) {
	si := NewSortedIndex(FromValue[uint8]())
	set(si, 1, 1)
	set(si, 2, 2)
	set(si, 3, 3)

	// errors
	_, _, err := si.MatchMany(query.FOpIn, "b", 1)
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

func TestIndex_Inverse(t *testing.T) {
	index := []struct {
		name  string
		index Index[uint8]
	}{
		{"sorted", NewSortedIndex(FromValue[uint8]())},
		{"range", NewRangeIndex(FromValue[uint8]())},
	}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			set(tt.index, 1, 1)
			set(tt.index, 2, 2)
			set(tt.index, 3, 3)
			set(tt.index, 4, 4)
			set(tt.index, 5, 5)

			allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3, 4, 5)

			bs, _, err := tt.index.Match(allIDs, query.FOpGt, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{2, 3, 4, 5}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpGe, 1)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1, 2, 3, 4, 5}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpLt, 5)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1, 2, 3, 4}, bs.ToSlice())

			bs, _, err = tt.index.Match(allIDs, query.FOpLe, 5)
			assert.NoError(t, err)
			assert.Equal(t, []uint32{1, 2, 3, 4, 5}, bs.ToSlice())
		})
	}
}

func TestStringIndex(t *testing.T) {
	ti := NewStringIndex(FromValue[string]()).AddTrigramIndex()

	set(ti, "abba", 1)
	set(ti, "acca", 2)
	set(ti, "bbba", 3)
	set(ti, "abxy", 4)

	allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3)

	// contains
	bs, _, _ := ti.Match(allIDs, query.FOpLike, "%bb%")
	assert.Equal(t, []uint32{1, 3}, bs.ToSlice())

	bs, _, _ = ti.Match(allIDs, query.FOpLike, "%nix%")
	assert.Equal(t, []uint32{}, bs.ToSlice())

	bs, _, _ = ti.Match(allIDs, query.FOpLike, "%acca%")
	assert.Equal(t, []uint32{2}, bs.ToSlice())

	// startsWith
	bs, _, _ = ti.Match(allIDs, query.FOpLike, "ab%")
	assert.Equal(t, []uint32{1, 4}, bs.ToSlice())

	// remove abba
	unSet(ti, "abba", 1)
	bs, _, _ = ti.Match(allIDs, query.FOpLike, "%bb%")
	assert.Equal(t, []uint32{3}, bs.ToSlice())
}

func TestStringIndex_Error(t *testing.T) {
	ti := NewStringIndex(FromValue[string]()).AddTrigramIndex()
	allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3)

	// contains
	_, _, err := ti.Match(allIDs, query.FilterOp{Name: "contains"}, "%bb%")
	assert.ErrorIs(t, InvalidOperationError{MapIndexName, 0}, err)

	// startsWith
	_, _, err = ti.Match(allIDs, query.FilterOp{Name: "startswith"}, "bb%")
	assert.ErrorIs(t, InvalidOperationError{MapIndexName, 0}, err)
}

func TestParserExt(t *testing.T) {
	fi := NewParserExt(
		NewRangeIndex(FromValue[uint8]()), func(s string) any {
			switch s {
			case "a":
				return 1
			case "b":
				return 2
			case "c":
				return 3
			case "d":
				return 4
			default:
				return 99
			}
		})

	set(fi, 1, 1)
	set(fi, 2, 2)
	set(fi, 3, 3)
	set(fi, 4, 4)

	rids, _ := fi.Equal("a")
	assert.Equal(t, []uint32{1}, rids.ToSlice())

	allIDs := lidx.NewRawIDsFrom[uint32](1, 2, 3, 4)
	rids, _, _ = fi.Match(allIDs, query.FOpGt, "a")
	assert.Equal(t, []uint32{2, 3, 4}, rids.ToSlice())

	rids, _, _ = fi.Match(allIDs, query.FOpGe, "d")
	assert.Equal(t, []uint32{4}, rids.ToSlice())

	rids, _, _ = fi.MatchMany(query.FOpIn, "a", "d")
	assert.Equal(t, []uint32{1, 4}, rids.ToSlice())

	rids, _, _ = fi.MatchMany(query.FOpBetween, "a", "d")
	assert.Equal(t, []uint32{1, 2, 3, 4}, rids.ToSlice())
}

func TestIndex_SliceValues(t *testing.T) {
	index := []struct {
		name  string
		index Index[[]uint8]
	}{
		{"range", NewRangeIndexSlice(FromValueSlice[uint8]())},
		{"map", NewMapIndexSlice(FromValueSlice[uint8]())},
		{"sorted", NewSortedIndexSlice(FromValueSlice[uint8]())},
	}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			tt.index.Set(&[]uint8{0, 3}, 0)
			tt.index.Set(&[]uint8{2, 3, 4}, 1)
			tt.index.Set(&[]uint8{2, 5}, 2)

			rids, _ := tt.index.Equal(0)
			assert.Equal(t, []uint32{0}, rids.ToSlice())

			rids, _ = tt.index.Equal(3)
			assert.Equal(t, []uint32{0, 1}, rids.ToSlice())

			rids, _ = tt.index.Equal(4)
			assert.Equal(t, []uint32{1}, rids.ToSlice())

			// not found
			rids, _ = tt.index.Equal(100)
			assert.Equal(t, []uint32{}, rids.ToSlice())

			rids, _, _ = tt.index.MatchMany(query.FOpIn, 3, 4)
			assert.Equal(t, []uint32{0, 1}, rids.ToSlice())

			rids, _, _ = tt.index.MatchMany(query.FOpIn, 6, 5)
			assert.Equal(t, []uint32{2}, rids.ToSlice())

			// not found
			rids, _, _ = tt.index.MatchMany(query.FOpIn, 100, 99)
			assert.Equal(t, []uint32{}, rids.ToSlice())
		})
	}
}

func TestIndex_SliceValues_More(t *testing.T) {
	index := []struct {
		name  string
		index Index[[]uint8]
	}{
		{"range", NewRangeIndexSlice(FromValueSlice[uint8]())},
		{"sorted", NewSortedIndexSlice(FromValueSlice[uint8]())},
	}

	for _, tt := range index {
		t.Run(tt.name, func(t *testing.T) {
			tt.index.Set(&[]uint8{0, 3}, 0)
			tt.index.Set(&[]uint8{2, 3, 4}, 1)
			tt.index.Set(&[]uint8{2, 5}, 2)

			allIDs := lidx.NewRawIDsFrom[uint32](0, 1, 2)

			rids, _, _ := tt.index.Match(allIDs, query.FOpGe, 2)
			assert.Equal(t, []uint32{0, 1, 2}, rids.ToSlice())

			rids, _, _ = tt.index.Match(allIDs, query.FOpLt, 4)
			assert.Equal(t, []uint32{0, 1, 2}, rids.ToSlice())

			// MatchMany
			rids, _, _ = tt.index.MatchMany(query.FOpBetween, 3, 4)
			assert.Equal(t, []uint32{0, 1}, rids.ToSlice())

			rids, _, _ = tt.index.MatchMany(query.FOpBetween, 5, 9)
			assert.Equal(t, []uint32{2}, rids.ToSlice())

			// not found
			rids, _, _ = tt.index.MatchMany(query.FOpBetween, 99, 102)
			assert.Equal(t, []uint32{}, rids.ToSlice())
		})
	}
}
