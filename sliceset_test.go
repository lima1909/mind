package mind

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
	assert.Equal(t, 2, sp.MaxSetIndex())
	assert.Equal(t, []uint16{1, 3, 42}, sp.ToSlice())

	assert.True(t, sp.UnSet(3))
	assert.False(t, sp.UnSet(99))
	assert.Equal(t, 2, sp.Len())
	assert.Equal(t, 1, sp.MaxSetIndex())
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

func TestSliceSet_Range(t *testing.T) {

	tests := []struct {
		name     string
		ss       *SliceSet[uint32]
		from     uint32
		to       uint32
		expected []uint32
	}{
		{
			name:     "Middle of set",
			ss:       NewSliceSetFrom[uint32](1, 2, 8, 42),
			from:     2,
			to:       8,
			expected: []uint32{2, 8},
		},
		{
			name:     "Last value",
			ss:       NewSliceSetFrom[uint32](0, 1, 2),
			from:     2,
			to:       2,
			expected: []uint32{2},
		},
		{
			name:     "Single bit range (Exact match)",
			ss:       NewSliceSetFrom[uint32](10, 20, 30),
			from:     20,
			to:       20,
			expected: []uint32{20},
		},
		{
			name:     "Empty Range (Nothing found)",
			ss:       NewSliceSetFrom[uint32](10, 20, 30),
			from:     11,
			to:       19,
			expected: nil,
		},
		{
			name: "Spanning Word Boundaries",
			// Word 0: bit 63 | Word 1: bit 64, 65 | Word 2: bit 130
			ss:       NewSliceSetFrom[uint32](0, 63, 64, 65, 130),
			from:     63,
			to:       100,
			expected: []uint32{63, 64, 65},
		},
		{
			name:     "From > To (Invalid range)",
			ss:       NewSliceSetFrom[uint32](1, 2, 3),
			from:     10,
			to:       5,
			expected: nil,
		},
		{
			name:     "Full Set Range",
			ss:       NewSliceSetFrom[uint32](5, 10),
			from:     0,
			to:       100,
			expected: []uint32{5, 10},
		},
		{
			name:     "Boundary: Bit at 0",
			ss:       NewSliceSetFrom[uint32](0, 1, 2),
			from:     0,
			to:       0,
			expected: []uint32{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var results []uint32
			tt.ss.Range(tt.from, tt.to, func(val uint32) bool {
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
