package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRawID_Base(t *testing.T) {
	sp := NewRawIDs[uint16]()
	sp.Set(42)
	sp.Set(3)
	sp.Set(1)

	assert.True(t, sp.IsSlice(), "small set should use SliceSet")
	assert.Equal(t, 3, sp.Count())
	assert.Equal(t, 1, sp.Min())
	assert.Equal(t, 42, sp.Max())
	assert.Equal(t, []uint16{1, 3, 42}, sp.ToSlice())

	assert.True(t, sp.Contains(3))
	assert.False(t, sp.Contains(99))

	assert.True(t, sp.UnSet(3))
	assert.False(t, sp.UnSet(99))
	assert.Equal(t, 2, sp.Count())
}

func TestRawID_DoubleAndOrderCheck(t *testing.T) {
	s := NewRawIDsFrom[uint16](3, 1, 0, 3, 1)
	assert.Equal(t, []uint16{0, 1, 3}, s.ToSlice())
}

func TestRawID_Empty(t *testing.T) {
	s := NewRawIDs[uint32]()
	assert.Equal(t, 0, s.Count())
	assert.Equal(t, 0, s.Len())
	assert.Equal(t, -1, s.Min())
	assert.Equal(t, -1, s.Max())
	assert.Equal(t, -1, s.MaxSetIndex())
	assert.False(t, s.Contains(0))
	assert.Equal(t, []uint32{}, s.ToSlice())
}

func TestRawID_PromotionToBitSet(t *testing.T) {
	s := NewRawIDs[uint32]()

	// Insert enough dense values to trigger promotion
	for i := uint32(0); i < 100; i++ {
		s.Set(i)
	}

	assert.True(t, s.IsBitSet(), "set with 100 dense elements should promote to BitSet")
	assert.Equal(t, 100, s.Count())
	assert.Equal(t, 0, s.Min())
	assert.Equal(t, 99, s.Max())

	// All values should still be accessible
	for i := uint32(0); i < 100; i++ {
		assert.True(t, s.Contains(i))
	}
}

func TestRawID_SparseDataStaysSlice(t *testing.T) {
	s := NewRawIDs[uint32]()

	// Insert sparse values (large gaps, few elements)
	for i := uint32(0); i < 30; i++ {
		s.Set(i * 1000) // values: 0, 1000, 2000, ..., 29000
	}

	assert.True(t, s.IsSlice(), "sparse data should stay as SliceSet")
	assert.Equal(t, 30, s.Count())
}

func TestRawID_NewFromWithPromotion(t *testing.T) {
	// Dense data should get promoted in constructor
	values := make([]uint32, 100)
	for i := range values {
		values[i] = uint32(i)
	}
	s := NewRawIDsFrom(values...)
	assert.True(t, s.IsBitSet(), "dense data in NewRawIDsFrom should promote")

	// Sparse data should stay as SliceSet
	sparseValues := []uint32{1, 1000, 2000, 30000, 50000}
	s2 := NewRawIDsFrom(sparseValues...)
	assert.True(t, s2.IsSlice(), "sparse data in NewRawIDsFrom should stay SliceSet")
}

func TestRawID_Copy(t *testing.T) {
	s := NewRawIDsFrom[uint16](1, 3, 42)
	c := s.Copy()

	assert.Equal(t, s.ToSlice(), c.ToSlice())

	// Mutating copy should not affect original
	c.Set(99)
	assert.False(t, s.Contains(99))
	assert.True(t, c.Contains(99))
}

func TestRawID_And(t *testing.T) {
	s1 := NewRawIDsFrom[uint16](42, 3, 1)
	s2 := NewRawIDsFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.And(s2)
	assert.Equal(t, []uint16{3}, result.ToSlice())

	// And with empty
	result = s1.Copy()
	empty := NewRawIDs[uint16]()
	result.And(empty)
	assert.Equal(t, []uint16{}, result.ToSlice())

	// Empty And non-empty
	empty = NewRawIDs[uint16]()
	empty.And(s2)
	assert.Equal(t, []uint16{}, empty.ToSlice())

	// Empty And empty
	empty = NewRawIDs[uint16]()
	empty.And(NewRawIDs[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestRawID_Or(t *testing.T) {
	s1 := NewRawIDsFrom[uint16](42, 3, 1)
	s2 := NewRawIDsFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.Or(s2)
	assert.Equal(t, []uint16{0, 1, 2, 3, 42}, result.ToSlice())

	// Or with empty
	result = s1.Copy()
	empty := NewRawIDs[uint16]()
	result.Or(empty)
	assert.Equal(t, []uint16{1, 3, 42}, result.ToSlice())

	// Empty Or non-empty
	empty = NewRawIDs[uint16]()
	empty.Or(s2)
	assert.Equal(t, []uint16{0, 2, 3}, empty.ToSlice())

	// Empty Or Empty
	empty = NewRawIDs[uint16]()
	empty.Or(NewRawIDs[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestRawID_Xor(t *testing.T) {
	s1 := NewRawIDsFrom[uint16](42, 3, 1)
	s2 := NewRawIDsFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.Xor(s2)
	assert.Equal(t, []uint16{0, 1, 2, 42}, result.ToSlice())

	result = s1.Copy()
	empty := NewRawIDs[uint16]()
	result.Xor(empty)
	assert.Equal(t, []uint16{1, 3, 42}, result.ToSlice())

	empty = NewRawIDs[uint16]()
	empty.Xor(s2)
	assert.Equal(t, []uint16{0, 2, 3}, empty.ToSlice())

	empty = NewRawIDs[uint16]()
	empty.Xor(NewRawIDs[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestRawID_AndNot(t *testing.T) {
	s1 := NewRawIDsFrom[uint16](42, 3, 1)
	s2 := NewRawIDsFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.AndNot(s2)
	assert.Equal(t, []uint16{1, 42}, result.ToSlice())

	result = s2.Copy()
	result.AndNot(s1)
	assert.Equal(t, []uint16{0, 2}, result.ToSlice())

	result = s1.Copy()
	empty := NewRawIDs[uint16]()
	result.AndNot(empty)
	assert.Equal(t, []uint16{1, 3, 42}, result.ToSlice())

	empty = NewRawIDs[uint16]()
	empty.AndNot(s2)
	assert.Equal(t, []uint16{}, empty.ToSlice())

	empty = NewRawIDs[uint16]()
	empty.AndNot(NewRawIDs[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestRawID_Values(t *testing.T) {
	s := NewRawIDsFrom[uint16](5, 3, 1, 42)
	var collected []uint16
	s.Values(func(v uint16) bool {
		collected = append(collected, v)
		return true
	})
	assert.Equal(t, []uint16{1, 3, 5, 42}, collected)

	// Early termination
	collected = nil
	s.Values(func(v uint16) bool {
		collected = append(collected, v)
		return len(collected) < 2
	})
	assert.Equal(t, 2, len(collected))
}

func TestRawID_MixedRepresentationOps(t *testing.T) {
	// Create a SliceSet-backed sparse set (small, sparse)
	sliceBacked := NewRawIDsFrom[uint32](1, 3, 5)
	assert.True(t, sliceBacked.IsSlice())

	// Create a BitSet-backed sparse set (dense, many elements)
	values := make([]uint32, 100)
	for i := range values {
		values[i] = uint32(i)
	}
	bitBacked := NewRawIDsFrom(values...)
	assert.True(t, bitBacked.IsBitSet())

	// And: slice ∩ bitset
	result := sliceBacked.Copy()
	result.And(bitBacked)
	assert.Equal(t, []uint32{1, 3, 5}, result.ToSlice())

	// Or: slice ∪ bitset
	result = sliceBacked.Copy()
	result.Or(bitBacked)
	assert.Equal(t, 100, result.Count())
	assert.True(t, result.Contains(1))
	assert.True(t, result.Contains(99))

	// AndNot: bitset \ slice
	result = bitBacked.Copy()
	result.AndNot(sliceBacked)
	assert.Equal(t, 97, result.Count())
	assert.False(t, result.Contains(1))
	assert.False(t, result.Contains(3))
	assert.False(t, result.Contains(5))
	assert.True(t, result.Contains(0))
	assert.True(t, result.Contains(2))
}

func TestRawID_DemotionAfterAnd(t *testing.T) {
	// Create two BitSet-backed sets
	values1 := make([]uint32, 100)
	for i := range values1 {
		values1[i] = uint32(i)
	}
	s1 := NewRawIDsFrom(values1...)
	assert.True(t, s1.IsBitSet())

	// Second set shares only 2 elements
	s2 := NewRawIDsFrom[uint32](5, 50)

	s1.And(s2)
	assert.Equal(t, []uint32{5, 50}, s1.ToSlice())
	// After And, the result is tiny and should demote to SliceSet
	assert.True(t, s1.IsSlice(), "small intersection should demote to SliceSet")
}

func TestRawID_Rebalance(t *testing.T) {
	// Create a BitSet-backed set, then remove most elements
	values := make([]uint32, 100)
	for i := range values {
		values[i] = uint32(i)
	}
	s := NewRawIDsFrom(values...)
	assert.True(t, s.IsBitSet())

	// Remove most elements
	for i := uint32(5); i < 100; i++ {
		s.UnSet(i)
	}
	assert.Equal(t, 5, s.Count())

	// Manual rebalance should demote
	s.Rebalance()
	assert.True(t, s.IsSlice(), "after removing most elements and rebalancing, should demote")
	assert.Equal(t, []uint32{0, 1, 2, 3, 4}, s.ToSlice())
}

func TestRawID_ToBitSet(t *testing.T) {
	s := NewRawIDsFrom[uint16](1, 3, 5)
	b := s.ToBitSet()
	assert.True(t, b.Contains(1))
	assert.True(t, b.Contains(3))
	assert.True(t, b.Contains(5))
	assert.Equal(t, 3, b.Count())
}

func TestRawID_ToSliceSet(t *testing.T) {
	s := NewRawIDsFrom[uint16](1, 3, 5)
	sl := s.ToSliceSet()
	assert.Equal(t, []uint16{1, 3, 5}, sl.ToSlice())
}

func TestRawID_WithCapacity(t *testing.T) {
	s := NewRawIDsWithCapacity[uint32](100)
	assert.Equal(t, 0, s.Count())
	s.Set(42)
	assert.Equal(t, 1, s.Count())
	assert.True(t, s.Contains(42))
}
