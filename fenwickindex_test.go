package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFenwickIndex_Base(t *testing.T) {
	idx := NewFenwickIndexWithMinValue(FromValue[int](), -10, 10)
	set := func(val int, lidx uint32) { idx.Set(&val, lidx) }
	set(-10, 1)
	set(-5, 2)
	set(-5, 3)
	set(0, 4)
	set(5, 5)
	set(10, 6)

	allIDs := NewRawIDsFrom[uint32](1, 2, 3, 4, 5, 6)

	t.Run("Operation: OpLe (<=)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpLe}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLe}, -20)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLe}, 20)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 4, 5, 6}, res.ToSlice())
	})

	t.Run("Operation: OpLt (<)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpLt}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLt}, -10)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())
	})

	t.Run("Operation: OpGt (>)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpGt}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{4, 5, 6}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpGt}, -15)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 4, 5, 6}, res.ToSlice())
	})

	t.Run("Operation: OpGe (>=)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpGe}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{2, 3, 4, 5, 6}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpGe}, 15)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())
	})

	t.Run("Operation: OpBetween", func(t *testing.T) {
		res, _, err := idx.MatchMany(FilterOp{Op: OpBetween}, -10, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3}, res.ToSlice())

		res, _, err = idx.MatchMany(FilterOp{Op: OpBetween}, -5, 5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{2, 3, 4, 5}, res.ToSlice())

		res, _, err = idx.MatchMany(FilterOp{Op: OpBetween}, 5, -5)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())

		res, _, err = idx.MatchMany(FilterOp{Op: OpBetween}, -50, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3}, res.ToSlice())
	})
}

func TestFenwickIndex_MinMax_negative(t *testing.T) {
	// max < min => switchde to: min: -100 and max: -1
	idx := NewFenwickIndexWithMinValue(FromValue[int](), -1, -100)
	set := func(val int, lidx uint32) { idx.Set(&val, lidx) }
	set(-10, 1)
	set(-5, 2)
	set(-5, 3)
	set(-1, 4)
	set(-100, 5)

	allIDs := NewRawIDsFrom[uint32](1, 2, 3, 4, 5)

	t.Run("Operation: OpLe (<=)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpLe}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 5}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLe}, -200)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLe}, -1)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 4, 5}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLe}, 100)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 4, 5}, res.ToSlice())
	})

	t.Run("Operation: OpLt (<)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpLt}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 5}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLt}, -100)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpLt}, 100)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 4, 5}, res.ToSlice())
	})

	t.Run("Operation: OpGt (>)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpGt}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{4}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpGt}, -100)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3, 4}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpGt}, 0)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{}, res.ToSlice())
	})

	t.Run("Operation: OpGe (>=)", func(t *testing.T) {
		res, _, err := idx.Match(allIDs, FilterOp{Op: OpGe}, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{2, 3, 4}, res.ToSlice())

		res, _, err = idx.Match(allIDs, FilterOp{Op: OpGe}, 15)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())
	})

	t.Run("Operation: OpBetween", func(t *testing.T) {
		res, _, err := idx.MatchMany(FilterOp{Op: OpBetween}, -10, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3}, res.ToSlice())

		res, _, err = idx.MatchMany(FilterOp{Op: OpBetween}, -5, 5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{2, 3, 4}, res.ToSlice())

		res, _, err = idx.MatchMany(FilterOp{Op: OpBetween}, 5, -5)
		assert.NoError(t, err)
		assert.True(t, res.IsEmpty())

		res, _, err = idx.MatchMany(FilterOp{Op: OpBetween}, -50, -5)
		assert.NoError(t, err)
		assert.Equal(t, []uint32{1, 2, 3}, res.ToSlice())
	})
}
