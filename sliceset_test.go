package mind

import (
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
