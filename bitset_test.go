package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitSet_Base(t *testing.T) {
	b := NewBitSet[uint8]()
	assert.False(t, b.Contains(0))

	b.Set(0)
	b.Set(1)
	b.Set(2)
	b.Set(2)
	b.Set(42)

	assert.True(t, b.Contains(0))

	assert.Equal(t, 4, b.Count())
	assert.Equal(t, 1, b.Len())
	assert.Equal(t, 0, b.Min())
	assert.Equal(t, 42, b.Max())
	assert.Equal(t, 0, b.MaxSetIndex())
	assert.True(t, b.Contains(2))

	b.UnSet(2)
	assert.False(t, b.Contains(2))
	assert.Equal(t, 3, b.Count())
	assert.Equal(t, 1, b.Len())

	b.UnSet(42)
	assert.Equal(t, 1, b.Max())

	_ = b.usedBytes()
}

func TestBitSet_ToBig(t *testing.T) {
	b := NewBitSet[uint32]()

	assert.Equal(t, -1, b.MaxSetIndex())

	assert.False(t, b.UnSet(40_000))
	assert.False(t, b.Contains(40_000))
}

func TestBitSet_Shrink(t *testing.T) {
	b := NewBitSet[uint16]()
	b.Set(1)
	b.Set(130)

	assert.Equal(t, 2, b.MaxSetIndex())

	assert.Equal(t, 2, b.Count())
	assert.Equal(t, 3, b.Len())
	assert.True(t, b.UnSet(130))

	b.Shrink()
	assert.Equal(t, 1, b.Count())
	assert.Equal(t, 1, b.Len())
	assert.Equal(t, 0, b.MaxSetIndex())
}

func TestBitSet_And(t *testing.T) {
	b1 := NewBitSetFrom[uint32](1, 2, 110, 2345)
	b2 := NewBitSetFrom[uint32](110)
	result := b1.Copy()
	result.And(b2)
	assert.Equal(t, []uint32{110}, result.ToSlice())

	b1 = NewBitSetFrom[uint32](110)
	b2 = NewBitSetFrom[uint32](1, 2, 110, 2345)
	result = b1.Copy()
	result.And(b2)
	assert.Equal(t, []uint32{110}, result.ToSlice())

	b0 := NewBitSet[uint32]()
	b0.And(b1)
	assert.Equal(t, 0, b0.Count())

	// b1 is removed
	b1.And(b0)
	assert.Equal(t, 0, b1.Count())
}

func TestBitSet_Or(t *testing.T) {
	b0 := NewBitSet[uint32]()
	b2 := NewBitSetFrom[uint32](110)
	b0.Or(b2)
	assert.Equal(t, []uint32{110}, b0.ToSlice())
}

func TestBitSet_Or1(t *testing.T) {
	b1 := NewBitSetFrom[uint32](1, 2, 110, 2345)
	b2 := NewBitSetFrom[uint32](110)
	result := b1.Copy()
	result.Or(b2)
	assert.Equal(t, []uint32{1, 2, 110, 2345}, result.ToSlice())

	b1 = NewBitSetFrom[uint32](110)
	b2 = NewBitSetFrom[uint32](1, 2, 110, 2345)
	result = b1.Copy()
	result.Or(b2)
	assert.Equal(t, []uint32{1, 2, 110, 2345}, result.ToSlice())

	b0 := NewBitSet[uint32]()
	result = b0.Copy()
	result.Or(b1)
	assert.Equal(t, []uint32{110}, result.ToSlice())

	// b1 is removed
	b1.Or(b0)
	assert.Equal(t, []uint32{110}, result.ToSlice())
}

func TestBitSet_Or2(t *testing.T) {
	b1 := NewBitSetFrom[uint32](1, 2, 3)
	b2 := NewBitSetFrom[uint32](4, 5, 6)
	result := b1.Copy()
	result.Or(b2)
	assert.Equal(t, []uint32{1, 2, 3, 4, 5, 6}, result.ToSlice())

	b1 = NewBitSetFrom[uint32](4, 5, 6)
	b2 = NewBitSetFrom[uint32](1, 2, 3)
	result = b1.Copy()
	result.Or(b2)
	assert.Equal(t, []uint32{1, 2, 3, 4, 5, 6}, result.ToSlice())
}

func TestBitSet_Xor(t *testing.T) {
	b1 := NewBitSetFrom[uint32](1, 2, 110, 2345)
	b2 := NewBitSetFrom[uint32](110)
	result := b1.Copy()
	result.Xor(b2)
	assert.Equal(t, []uint32{1, 2, 2345}, result.ToSlice())

	// shrinked?
	// assert.Equal(t, 1, result.Count())
	// assert.Equal(t, 2, len(result.data))

	b1 = NewBitSetFrom[uint32](110)
	b2 = NewBitSetFrom[uint32](1, 2, 110, 2345)
	result = b1.Copy()
	result.Xor(b2)
	assert.Equal(t, []uint32{1, 2, 2345}, result.ToSlice())

	// no overlap
	b3 := NewBitSetFrom[uint32](3)
	result = b3.Copy()
	result.Xor(b1)
	assert.Equal(t, []uint32{3, 110}, result.ToSlice())

	b0 := NewBitSet[uint32]()
	result = b0.Copy()
	result.Or(b1)
	assert.Equal(t, []uint32{110}, result.ToSlice())

	// b1 is removed
	b1.Or(b0)
	assert.Equal(t, []uint32{110}, result.ToSlice())
}

func TestBitSet_AndNot(t *testing.T) {
	b1 := NewBitSetFrom[uint64](1, 2, 110, 2345)
	b2 := NewBitSetFrom[uint64](110, 2)
	result := b1.Copy()
	result.AndNot(b2)
	assert.Equal(t, []uint64{1, 2345}, result.ToSlice())

	b1 = NewBitSetFrom[uint64](110, 2)
	b2 = NewBitSetFrom[uint64](1, 2, 110, 2345)
	result = b1.Copy()
	result.AndNot(b2)
	result.Shrink()
	assert.Equal(t, NewBitSetFrom[uint64](), result)

	b0 := NewBitSet[uint64]()
	b0.AndNot(b1)
	assert.Equal(t, 0, b0.Count())

	// b1 is removed
	b1.AndNot(b0)
	assert.Equal(t, []uint64{2, 110}, b1.ToSlice())
}

func TestBitSet_MinMax(t *testing.T) {
	b := NewBitSet[uint8]()
	b.Set(0)
	b.Set(1)
	b.Set(5)
	b.Set(52)
	b.Set(67)
	b.Set(130)

	assert.Equal(t, 0, b.Min())
	assert.Equal(t, 130, b.Max())
	// 0, 1, 2
	assert.Equal(t, 2, b.MaxSetIndex())

	b.UnSet(0)
	b.UnSet(130)
	assert.Equal(t, 1, b.Min())
	assert.Equal(t, 67, b.Max())
	// 0, 1
	assert.Equal(t, 1, b.MaxSetIndex())
}

func TestBitSet_ValuesIter(t *testing.T) {
	b := NewBitSet[uint8]()
	b.Set(2)
	b.Set(1)
	b.Set(2)
	b.Set(0)
	b.Set(142)

	values := make([]uint8, 0)
	b.Values(func(v uint8) bool {
		values = append(values, v)
		return true
	})

	assert.Equal(t, []uint8{0, 1, 2, 142}, values)
}

func TestBitSet_Range(t *testing.T) {

	tests := []struct {
		name     string
		bs       *BitSet[uint32]
		from     uint32
		to       uint32
		expected []uint32
	}{
		{
			name:     "Middle of set",
			bs:       NewBitSetFrom[uint32](1, 2, 8, 42),
			from:     2,
			to:       8,
			expected: []uint32{2, 8},
		},
		{
			name:     "Last value",
			bs:       NewBitSetFrom[uint32](0, 1, 2),
			from:     2,
			to:       2,
			expected: []uint32{2},
		},
		{
			name:     "Single bit range (Exact match)",
			bs:       NewBitSetFrom[uint32](10, 20, 30),
			from:     20,
			to:       20,
			expected: []uint32{20},
		},
		{
			name:     "Empty Range (Nothing found)",
			bs:       NewBitSetFrom[uint32](10, 20, 30),
			from:     11,
			to:       19,
			expected: nil,
		},
		{
			name: "Spanning Word Boundaries",
			// Word 0: bit 63 | Word 1: bit 64, 65 | Word 2: bit 130
			bs:       NewBitSetFrom[uint32](0, 63, 64, 65, 130),
			from:     63,
			to:       100,
			expected: []uint32{63, 64, 65},
		},
		{
			name:     "From > To (Invalid range)",
			bs:       NewBitSetFrom[uint32](1, 2, 3),
			from:     10,
			to:       5,
			expected: nil,
		},
		{
			name:     "Full Set Range",
			bs:       NewBitSetFrom[uint32](5, 10),
			from:     0,
			to:       100,
			expected: []uint32{5, 10},
		},
		{
			name:     "Boundary: Bit at 0",
			bs:       NewBitSetFrom[uint32](0, 1, 2),
			from:     0,
			to:       0,
			expected: []uint32{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var results []uint32
			tt.bs.Range(tt.from, tt.to, func(val uint32) bool {
				results = append(results, val)
				return true
			})

			if len(results) != len(tt.expected) {
				t.Fatalf("expected %d results, got %d", len(tt.expected), len(results))
			}

			for i := range results {
				if results[i] != tt.expected[i] {
					t.Errorf("at result %d: expected %+v, got %+v", i, tt.expected[i], results[i])
				}
			}
		})
	}
}

func TestBitSet_Clear(t *testing.T) {
	b := NewBitSet[uint8]()
	b.Set(0)
	b.Set(1)
	b.Set(68)

	assert.True(t, b.Contains(1))
	assert.Equal(t, 3, b.Count())

	b.Clear()
	assert.False(t, b.Contains(1))
	assert.Equal(t, 0, b.Count())
	assert.Equal(t, 0, len(b.data))
	assert.Equal(t, 2, cap(b.data))
}
