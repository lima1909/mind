package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func set[T any](idx Index[T], t T, r uint32)   { idx.Set(&t, r) }
func unSet[T any](idx Index[T], t T, r uint32) { idx.UnSet(&t, r) }

func fieldIndexMapFn[T any](mi Index[T]) FilterByName {
	return func(fieldName string) (Filter, error) {
		if fieldName == "val" {
			return mi, nil
		}

		return nil, InvalidNameError{fieldName}
	}
}

func TestMapIndex_UnSet(t *testing.T) {
	mi := NewMapIndex(FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	// check all values are correct
	bs, err := mi.Match(FOpEq, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	bs, err = mi.Match(FOpEq, 3)
	assert.NoError(t, err)
	assert.Equal(t, 2, bs.Count())
	bs, err = mi.Match(FOpEq, 42)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())

	// remove the last one: 42
	unSet(mi, 42, 42)
	bs, err = mi.Match(FOpEq, 42)
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())

	// remove value 3
	unSet(mi, 3, 3)
	bs, err = mi.Match(FOpEq, 3)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	unSet(mi, 3, 5)
	bs, err = mi.Match(FOpEq, 3)
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())

	// for value 1 is no row 99, no deletion (ignored)
	unSet(mi, 1, 99)
	bs, err = mi.Match(FOpEq, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())

	// remove value 1
	unSet(mi, 1, 1)
	bs, err = mi.Match(FOpEq, 1)
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())
}

func TestMapIndex_Get(t *testing.T) {
	mi := NewMapIndex(FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	bs, _ := mi.Match(FOpEq, 1)
	assert.Equal(t, NewBitSetFrom[uint32](1), bs)
	bs, _ = mi.Match(FOpEq, 3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())

	// not found
	bs, err := mi.Match(FOpEq, 7)
	assert.NoError(t, err)
	assert.True(t, bs.IsEmpty())

	// invalid relation
	_, err = mi.Match(FOpGt, 1)
	assert.ErrorIs(t, InvalidOperationError{MapIndexName, OpGt}, err)
}

func TestMapIndex_Query(t *testing.T) {
	mi := NewMapIndex(FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	result, canMutate, err := Eq("val", 3)(fi, nil)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{3, 5}, result.ToSlice())

	// repeat the Eq with the same paramter, to check the result BitSet is not changed
	result, _, err = Eq("val", 3)(fi, nil)
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 5}, result.ToSlice())

	// not found
	result, _, err = Eq("val", 99)(fi, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Count())

	// invalid field
	result, _, err = Eq("bad", 1)(fi, nil)
	assert.ErrorIs(t, InvalidNameError{"bad"}, err)
	assert.Nil(t, result)

	// OR
	result, canMutate, err = Or(Eq("val", 3), Eq("val", 42), Eq("val", 1))(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5, 42}, result.ToSlice())
	// three ORs
	result, canMutate, err = Or(Eq("val", 3), Eq("val", 1))(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5}, result.ToSlice())

	// AND
	result, canMutate, err = And(Eq("val", 3), Not(Eq("val", 3)))(fi, NewBitSetFrom[uint32](1, 3, 5, 42))
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{}, result.ToSlice())
	// three Ands
	result, canMutate, err = And(Eq("val", 3), Eq("val", 3), Eq("val", 3))(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{3, 5}, result.ToSlice())

	// combine OR and AND
	result, canMutate, err = Or(Eq("val", 1), And(Eq("val", 3), Eq("val", 3)))(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5}, result.ToSlice())

	// after and | or, to check the original BitSet is not changed
	bs, _ := mi.Match(FOpEq, 1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Match(FOpEq, 42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Match(FOpEq, 3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestMapIndex_Query_Not(t *testing.T) {
	mi := NewMapIndex(FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	allIDs := NewBitSetFrom[uint32](1, 3, 5, 42)

	// Not
	result, canMutate, err := Not(Eq("val", 3))(fi, allIDs)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// NotEq
	result, canMutate, err = NotEq("val", 3)(fi, allIDs)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// after and | or, to check the original BitSet is not changed
	bs, _ := mi.Match(FOpEq, 1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Match(FOpEq, 42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Match(FOpEq, 3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestMapIndex_Query_In(t *testing.T) {
	mi := NewMapIndex(FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	// In empty
	result, canMutate, err := In("val")(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{}, result.ToSlice())

	// In one
	result, canMutate, err = In("val", 1)(fi, nil)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{1}, result.ToSlice())

	// In many
	result, canMutate, err = In("val", 42, 1)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// after and | or, to check the original BitSet is not changed
	bs, _ := mi.Match(FOpEq, 1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Match(FOpEq, 42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Match(FOpEq, 3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestMapIndex_QueryAll(t *testing.T) {
	mi := NewMapIndex(FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)
	result, canMutate, err := All()(fi, NewBitSetFrom[uint32](1, 3, 5, 42))
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5, 42}, result.ToSlice())
}

func TestSortedIndex_StringStartsWith(t *testing.T) {
	mi := NewSortedIndex(FromValue[string]())
	set(mi, "app", 1)
	set(mi, "no", 3)
	set(mi, "appl", 5)
	set(mi, "appx", 42)

	fi := fieldIndexMapFn(mi)
	allIDs := NewBitSetFrom[uint32](1, 3, 5, 42)

	result, canMutate, err := WithPrefix("val", "not found")(fi, allIDs)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{}, result.ToSlice())

	result, canMutate, err = WithPrefix("val", "")(fi, allIDs)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5, 42}, result.ToSlice())

	result, canMutate, err = WithPrefix("val", "no")(fi, allIDs)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{3}, result.ToSlice())

	result, canMutate, err = WithPrefix("val", "app")(fi, allIDs)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{1, 5, 42}, result.ToSlice())
}
