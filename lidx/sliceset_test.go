package lidx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceSet_Base(t *testing.T) {
	sp := NewSliceSet[uint16]()
	sp.Set(42)
	sp.Set(3)
	sp.Set(1)

	assert.Equal(t, 3, sp.Count())
	assert.Equal(t, 3, sp.Len())
	assert.Equal(t, 1, sp.Min())
	assert.Equal(t, 42, sp.Max())
	assert.Equal(t, 2, sp.MaxIndex())
	assert.Equal(t, []uint16{1, 3, 42}, sp.ToSlice())

	assert.True(t, sp.UnSet(3))
	assert.False(t, sp.UnSet(99))
	assert.Equal(t, 2, sp.Len())
	assert.Equal(t, 1, sp.MaxIndex())
}
func TestSliceSet_DoubleAndOrderCheck(t *testing.T) {
	s := NewSliceSetFrom[uint16](3, 1, 0, 3, 1)
	assert.Equal(t, []uint16{0, 1, 3}, s.ToSlice())
}

func TestSliceSet_And(t *testing.T) {
	s1 := NewSliceSetFrom[uint16](42, 3, 1)
	s2 := NewSliceSetFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.And(s2)
	assert.Equal(t, []uint16{3}, result.ToSlice())

	s1.And(s1)
	assert.Equal(t, []uint16{1, 3, 42}, s1.ToSlice())

	result = s1.Copy()
	empty := NewSliceSet[uint16]()
	result.And(empty)
	assert.Equal(t, []uint16{}, result.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.And(s2)
	assert.Equal(t, []uint16{}, empty.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.And(NewSliceSet[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestSliceSet_Or(t *testing.T) {
	s1 := NewSliceSetFrom[uint16](42, 3, 1)
	s2 := NewSliceSetFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.Or(s2)
	assert.Equal(t, []uint16{0, 1, 2, 3, 42}, result.ToSlice())

	result = s1.Copy()
	empty := NewSliceSet[uint16]()
	result.Or(empty)
	assert.Equal(t, []uint16{1, 3, 42}, result.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.Or(s2)
	assert.Equal(t, []uint16{0, 2, 3}, empty.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.Or(NewSliceSet[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestSliceSet_Xor(t *testing.T) {
	s1 := NewSliceSetFrom[uint16](42, 3, 1)
	s2 := NewSliceSetFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.Xor(s2)
	assert.Equal(t, []uint16{0, 1, 2, 42}, result.ToSlice())

	result = s1.Copy()
	empty := NewSliceSet[uint16]()
	result.Xor(empty)
	assert.Equal(t, []uint16{1, 3, 42}, result.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.Xor(s2)
	assert.Equal(t, []uint16{0, 2, 3}, empty.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.Xor(NewSliceSet[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestSliceSet_AndNot(t *testing.T) {
	s1 := NewSliceSetFrom[uint16](42, 3, 1)
	s2 := NewSliceSetFrom[uint16](2, 3, 0)

	result := s1.Copy()
	result.AndNot(s2)
	assert.Equal(t, []uint16{1, 42}, result.ToSlice())

	result = s2.Copy()
	result.AndNot(s1)
	assert.Equal(t, []uint16{0, 2}, result.ToSlice())

	result = s1.Copy()
	empty := NewSliceSet[uint16]()
	result.AndNot(empty)
	assert.Equal(t, []uint16{1, 3, 42}, result.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.AndNot(s2)
	assert.Equal(t, []uint16{}, empty.ToSlice())

	empty = NewSliceSet[uint16]()
	empty.AndNot(NewSliceSet[uint16]())
	assert.Equal(t, []uint16{}, empty.ToSlice())
}

func TestSliceSet_ValueOnIndex(t *testing.T) {

	ss := NewSliceSetFrom[uint32](1, 2, 8, 42, 1028)

	tests := []struct {
		index    uint32
		found    bool
		expected uint32
	}{
		{
			// first
			index:    0,
			found:    true,
			expected: 1,
		},
		{
			// middle
			index:    2,
			found:    true,
			expected: 8,
		},
		{
			// end
			index:    4,
			found:    true,
			expected: 1028,
		},
		{
			// to big, not found
			index:    1000,
			found:    false,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("index_%d", tt.index), func(t *testing.T) {
			val, found := ss.ValueOnIndex(tt.index)
			assert.Equal(t, tt.found, found)
			assert.Equal(t, tt.expected, val)
		})
	}
}

// slice‑shift bug
// when Values removes two consecutive false positives.
func TestSliceSet_ValuesUnSetIter(t *testing.T) {
	ss := NewSliceSetFrom[uint32](0, 1, 2)
	// Remove the first two IDs (adjacent) during Values iteration.
	ss.Removes(func(id uint32) bool { return id == 0 || id == 1 })
	assert.Equal(t, []uint32{2}, ss.ToSlice())
}

func TestSliceSet_Removes(t *testing.T) {
	tests := []struct {
		name   string
		in     []uint32
		remove func(uint32) bool
		want   []uint32
	}{
		{"empty", nil, func(uint32) bool { return true }, []uint32{}},
		{"remove none", []uint32{1, 2, 3}, func(uint32) bool { return false }, []uint32{1, 2, 3}},
		{"remove all", []uint32{1, 2, 3}, func(uint32) bool { return true }, []uint32{}},
		{"remove first", []uint32{1, 2, 3}, func(v uint32) bool { return v == 1 }, []uint32{2, 3}},
		{"remove middle", []uint32{1, 2, 3}, func(v uint32) bool { return v == 2 }, []uint32{1, 3}},
		{"remove last", []uint32{1, 2, 3}, func(v uint32) bool { return v == 3 }, []uint32{1, 2}},
		{"remove even", []uint32{1, 2, 3, 4, 5, 6}, func(v uint32) bool { return v%2 == 0 }, []uint32{1, 3, 5}},
		{"sparse", []uint32{5, 100, 4096, 9999}, func(v uint32) bool { return v > 1000 }, []uint32{5, 100}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSliceSetFrom(tt.in...)
			s.Removes(tt.remove)

			assert.Equal(t, tt.want, s.ToSlice())
			assert.Equal(t, len(tt.want), s.Count())
			// the set must stay sorted, Contains relies on binary search
			for _, v := range tt.want {
				assert.True(t, s.Contains(v), "Contains(%d) failed after Removes", v)
			}
		})
	}
}

// Or with an empty receiver must COPY, not alias. Otherwise a later in-place
// mutation of the result corrupts the source, which is typically an index bucket.
func TestSliceSet_Or_MustNotAliasOther(t *testing.T) {
	empty := NewSliceSet[uint32]()
	src := NewSliceSetFrom[uint32](10, 20, 30)

	empty.Or(src)
	assert.Equal(t, []uint32{10, 20, 30}, empty.ToSlice())

	// mutate the result in every way that writes into the backing array
	empty.UnSet(20)
	assert.Equal(t, []uint32{10, 30}, empty.ToSlice())

	// src must be untouched
	assert.Equal(t, []uint32{10, 20, 30}, src.ToSlice(), "Or aliased the source slice")
}

func TestSliceSet_Or_MustNotAliasOther_AndInPlace(t *testing.T) {
	empty := NewSliceSet[uint32]()
	src := NewSliceSetFrom[uint32](0, 1, 2, 15, 29)

	empty.Or(src)
	// And compacts survivors to the left, which is what writes into the array
	empty.And(NewSliceSetFrom[uint32](15))

	assert.Equal(t, []uint32{15}, empty.ToSlice())
	assert.Equal(t, []uint32{0, 1, 2, 15, 29}, src.ToSlice(), "And corrupted the aliased source")
}

func TestSliceSet_ValuesBatch(t *testing.T) {
	t.Run("0", func(t *testing.T) {
		ss := NewSliceSet[uint32]()
		count := 0
		ss.ValuesBatch(func(v []uint32) bool {
			count += len(v)
			return true
		})
		assert.Equal(t, 0, count)
	})

	t.Run("250", func(t *testing.T) {
		ss := NewSliceSet[uint32]()
		for i := range 250 {
			ss.Set(uint32(i))
		}
		count := 0
		ss.ValuesBatch(func(v []uint32) bool {
			count += len(v)
			return true
		})
		assert.Equal(t, 250, count)
	})

	t.Run("500", func(t *testing.T) {
		ss := NewSliceSet[uint32]()
		for i := range 500 {
			ss.Set(uint32(i))
		}
		count := 0
		ss.ValuesBatch(func(v []uint32) bool {
			count += len(v)
			return true
		})
		assert.Equal(t, 500, count)
	})

	t.Run("1501", func(t *testing.T) {
		ss := NewSliceSet[uint32]()
		for i := range 1501 {
			ss.Set(uint32(i))
		}
		count := 0
		ss.ValuesBatch(func(v []uint32) bool {
			count += len(v)
			return true
		})
		assert.Equal(t, 1501, count)
	})

	t.Run("1600 every second", func(t *testing.T) {
		ss := NewSliceSet[uint32]()
		for i := range 1600 {
			if i%2 == 0 {
				ss.Set(uint32(i))
			}
		}
		count := 0
		ss.ValuesBatch(func(v []uint32) bool {
			count += len(v)
			return true
		})
		assert.Equal(t, 800, count)
	})

}

func BenchmarkSliceSet_And(b *testing.B) {
	bs1 := NewSliceSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%3 == 0 {
			bs1.Set(uint32(i))
		}
	}
	bs2 := NewSliceSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%6 == 0 {
			bs2.Set(uint32(i))
		}
	}
	b.ResetTimer()

	for b.Loop() {
		r := bs2.Copy()
		r.And(bs1)
		assert.Equal(b, 500_000, r.Count())
	}
}

func BenchmarkSliceSet_Or(b *testing.B) {
	bs1 := NewSliceSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%3 == 0 {
			bs1.Set(uint32(i))
		}
	}
	bs2 := NewSliceSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%6 == 0 {
			bs2.Set(uint32(i))
		}
	}
	b.ResetTimer()

	for b.Loop() {
		r := bs2.Copy()
		r.Or(bs1)
		assert.Equal(b, count/3, r.Count())
	}
}

func BenchmarkSLiceSet_ValuesBatchIter(b *testing.B) {
	bs := NewSliceSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		c := 0
		bs.ValuesBatch(func(v []uint32) bool {
			c += len(v)
			return true
		})
		assert.Equal(b, count, c)

	}
}
