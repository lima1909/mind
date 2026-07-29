package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lima1909/mind/errr"
	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/lidx"
	"github.com/lima1909/mind/query"
)

func set[T any](idx index.Index[T], t T, r uint32)   { idx.Set(&t, r) }
func unSet[T any](idx index.Index[T], t T, r uint32) { idx.UnSet(&t, r) }

func fieldIndexMapFn[T any](mi index.Index[T]) query.FilterByName {
	return func(fieldName string) (query.Filter, error) {
		if fieldName == "val" {
			return mi, nil
		}

		return nil, errr.InvalidNameError{FieldName: fieldName}
	}
}

func TestMapIndex_UnSet(t *testing.T) {
	mi := index.NewMapIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	// check all values are correct
	bs, err := mi.Equal(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	bs, err = mi.Equal(3)
	assert.NoError(t, err)
	assert.Equal(t, 2, bs.Count())
	bs, err = mi.Equal(42)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())

	// remove the last one: 42
	unSet(mi, 42, 42)
	bs, err = mi.Equal(42)
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())

	// remove value 3
	unSet(mi, 3, 3)
	bs, err = mi.Equal(3)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())
	unSet(mi, 3, 5)
	bs, err = mi.Equal(3)
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())

	// for value 1 is no row 99, no deletion (ignored)
	unSet(mi, 1, 99)
	bs, err = mi.Equal(1)
	assert.NoError(t, err)
	assert.Equal(t, 1, bs.Count())

	// remove value 1
	unSet(mi, 1, 1)
	bs, err = mi.Equal(1)
	assert.NoError(t, err)
	assert.Equal(t, 0, bs.Count())
}

func TestMapIndex_Get(t *testing.T) {
	mi := index.NewMapIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	bs, _ := mi.Equal(1)
	assert.Equal(t, lidx.NewRawIDsFrom[uint32](1), bs)
	bs, _ = mi.Equal(3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())

	// not found
	bs, err := mi.Equal(7)
	assert.NoError(t, err)
	assert.True(t, bs.IsEmpty())

	// invalid relation
	_, _, err = mi.Match(nil, query.FOpGt, 1)
	assert.ErrorIs(t, errr.InvalidOperationError{IndexName: index.MapIndexName, Op: query.OpGt}, err)
}

func TestMapIndex_Query(t *testing.T) {
	mi := index.NewMapIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	result, canMutate, err := query.Eq("val", 3).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{3, 5}, result.ToSlice())

	// repeat the Eq with the same paramter, to check the result RawIDs is not changed
	result, _, err = query.Eq("val", 3).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.Equal(t, []uint32{3, 5}, result.ToSlice())

	// not found
	result, _, err = query.Eq("val", 99).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, result.Count())

	// invalid field
	result, _, err = query.Eq("bad", 1).Compile(nil)(fi, nil)
	assert.ErrorIs(t, errr.InvalidNameError{FieldName: "bad"}, err)
	assert.Nil(t, result)

	// OR
	result, canMutate, err = query.Or(query.Or(query.Eq("val", 3), query.Eq("val", 42)), query.Eq("val", 1)).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5, 42}, result.ToSlice())
	// three ORs
	result, canMutate, err = query.Or(query.Eq("val", 3), query.Eq("val", 1)).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5}, result.ToSlice())

	// AND
	result, canMutate, err = query.And(query.Eq("val", 3), query.Not(query.Eq("val", 3))).Compile(nil)(fi, lidx.NewRawIDsFrom[uint32](1, 3, 5, 42))
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{}, result.ToSlice())
	// three Ands
	result, canMutate, err = query.And(query.And(query.Eq("val", 3), query.Eq("val", 3)), query.Eq("val", 3)).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{3, 5}, result.ToSlice())

	// combine OR and AND
	result, canMutate, err = query.Or(query.Eq("val", 1), query.And(query.Eq("val", 3), query.Eq("val", 3))).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5}, result.ToSlice())

	// after and | or, to check the original RawIDs is not changed
	bs, _ := mi.Equal(1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Equal(42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Equal(3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestMapIndex_Query_Not(t *testing.T) {
	mi := index.NewMapIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	allIDs := lidx.NewRawIDsFrom[uint32](1, 3, 5, 42)

	// Not
	result, canMutate, err := query.Not(query.Eq("val", 3)).Compile(nil)(fi, allIDs)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// NotEq
	result, canMutate, err = query.NotEq("val", 3).Optimize().Compile(nil)(fi, allIDs)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// after and | or, to check the original RawIDs is not changed
	bs, _ := mi.Equal(1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Equal(42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Equal(3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestSortedIndex_Query_Not(t *testing.T) {
	mi := index.NewSortedIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	allIDs := lidx.NewRawIDsFrom[uint32](1, 3, 5, 42)

	// Not
	result, canMutate, err := query.Not(query.Eq("val", 3)).Compile(nil)(fi, allIDs)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// NotEq
	result, canMutate, err = query.NotEq("val", 3).Optimize().Compile(nil)(fi, allIDs)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// after and | or, to check the original RawIDs is not changed
	bs, _ := mi.Equal(1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Equal(42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Equal(3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestMapIndex_Query_In(t *testing.T) {
	mi := index.NewMapIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)

	// In empty
	result, canMutate, err := query.In("val").Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{}, result.ToSlice())

	// In one
	result, canMutate, err = query.In("val", 1).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{1}, result.ToSlice())

	// In many
	result, canMutate, err = query.In("val", 42, 1).Compile(nil)(fi, nil)
	assert.NoError(t, err)
	assert.True(t, canMutate)
	assert.Equal(t, []uint32{1, 42}, result.ToSlice())

	// after and | or, to check the original RawIDs is not changed
	bs, _ := mi.Equal(1)
	assert.Equal(t, []uint32{1}, bs.ToSlice())
	bs, _ = mi.Equal(42)
	assert.Equal(t, []uint32{42}, bs.ToSlice())
	bs, _ = mi.Equal(3)
	assert.Equal(t, []uint32{3, 5}, bs.ToSlice())
}

func TestMapIndex_QueryAll(t *testing.T) {
	mi := index.NewMapIndex(index.FromValue[int]())
	set(mi, 1, 1)
	set(mi, 3, 3)
	set(mi, 3, 5)
	set(mi, 42, 42)

	fi := fieldIndexMapFn(mi)
	result, canMutate, err := query.All().Compile(nil)(fi, lidx.NewRawIDsFrom[uint32](1, 3, 5, 42))
	assert.NoError(t, err)
	assert.False(t, canMutate)
	assert.Equal(t, []uint32{1, 3, 5, 42}, result.ToSlice())
}
