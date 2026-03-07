package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFreeList_Base(t *testing.T) {
	l := NewFreeList[string]()
	assert.Equal(t, 0, l.Insert("a"))
	assert.Equal(t, 1, l.Insert("b"))
	assert.Equal(t, 2, l.Insert("c"))
	assert.Equal(t, 3, l.Count())

	val, found := l.Get(1)
	assert.True(t, found)
	assert.Equal(t, "b", val)

	assert.False(t, l.Remove(100))
	assert.True(t, l.Remove(1))

	val, found = l.Get(1)
	assert.False(t, found)
	assert.Equal(t, "", val)

	l.Insert("z")
	val, found = l.Get(1)
	assert.True(t, found)
	assert.Equal(t, "z", val)
}

func TestFreeList_Update(t *testing.T) {
	l := NewFreeList[string]()
	assert.Equal(t, 0, l.Insert("a"))
	assert.Equal(t, 1, l.Insert("b"))
	assert.Equal(t, 2, l.Insert("c"))
	assert.Equal(t, 3, l.Count())

	old, ok := l.Set(1, "z")
	assert.True(t, ok)
	assert.Equal(t, "b", old)

	// index to big
	_, ok = l.Set(100, "z")
	assert.False(t, ok)
	// negative index
	_, ok = l.Set(-100, "z")
	assert.False(t, ok)

	// index not found
	assert.True(t, l.Remove(1))
	assert.Equal(t, 2, l.Count())
	_, ok = l.Set(1, "z")
	assert.False(t, ok)

}

func TestFreeList_CompactUnstable(t *testing.T) {
	l := NewFreeList[string]()
	l.Insert("a")
	l.Insert("b")
	l.Insert("c")
	l.Insert("d")
	l.Insert("e")
	l.Insert("f")
	assert.Equal(t, 6, l.Count())

	l.Remove(1) // b
	l.Remove(2) // c
	l.Remove(4) // e
	assert.Equal(t, 3, l.Count())

	l.CompactUnstable()
	assert.Equal(t, 3, len(l.slots))
	assert.Equal(t, 3, l.Count())

	val, found := l.Get(0)
	assert.True(t, found)
	assert.Equal(t, "a", val)

	val, found = l.Get(1)
	assert.True(t, found)
	assert.Equal(t, "d", val)

	val, found = l.Get(2)
	assert.True(t, found)
	assert.Equal(t, "f", val)
}

func TestFreeList_CompactLinear(t *testing.T) {
	l := NewFreeList[string]()
	l.Insert("a")
	l.Insert("b")
	l.Insert("c")
	l.Insert("d")
	l.Insert("e")
	l.Insert("f")
	assert.Equal(t, 6, l.Count())

	l.Remove(1) // b
	l.Remove(2) // c
	l.Remove(4) // e
	assert.Equal(t, 3, l.Count())

	removed := make([]int, 0)
	l.CompactLinear(func(oldIndex, newIndex int) {
		removed = append(removed, oldIndex)
	})
	// the index 0 is not moved
	assert.Equal(t, []int{3, 5}, removed)
	assert.Equal(t, 3, len(l.slots))

	val, found := l.Get(0)
	assert.True(t, found)
	assert.Equal(t, "a", val)

	val, found = l.Get(1)
	assert.True(t, found)
	assert.Equal(t, "d", val)

	val, found = l.Get(2)
	assert.True(t, found)
	assert.Equal(t, "f", val)
}

func TestFreeList_Iter(t *testing.T) {
	l := NewFreeList[string]()
	assert.Equal(t, 0, l.Insert("a"))
	assert.Equal(t, 1, l.Insert("b"))
	assert.Equal(t, 2, l.Insert("c"))

	for idx, item := range l.Iter() {
		switch idx {
		case 0:
			assert.Equal(t, "a", item)
		case 1:
			assert.Equal(t, "b", item)
		case 2:
			assert.Equal(t, "c", item)
		default:
			assert.Failf(t, "invalid", "idx: %v", idx)
		}
	}

	// remove one item in the middle
	assert.True(t, l.Remove(1))
	for idx, item := range l.Iter() {
		switch idx {
		case 0:
			assert.Equal(t, "a", item)
		case 2:
			assert.Equal(t, "c", item)
		default:
			assert.Failf(t, "invalid", "idx: %v", idx)
		}
	}
}

func TestFreeList_Filter(t *testing.T) {
	l := NewFreeList[string]()
	assert.Equal(t, 0, l.Insert("a"))
	assert.Equal(t, 1, l.Insert("b"))
	assert.Equal(t, 2, l.Insert("c"))
	assert.Equal(t, 3, l.Insert("a"))
	assert.Equal(t, 4, l.Insert("b"))
	assert.Equal(t, 5, l.Insert("c"))

	result := make([]int, 0, 2)
	l.filter(func(item *string) bool {
		return *item == "a"
	}, func(idx int) bool {
		result = append(result, idx)
		return true
	})

	assert.Equal(t, []int{0, 3}, result)
}

func TestFreeList_FilterBS(t *testing.T) {
	l := NewFreeList[string]()
	assert.Equal(t, 0, l.Insert("a"))
	assert.Equal(t, 1, l.Insert("b"))
	assert.Equal(t, 2, l.Insert("c"))
	assert.Equal(t, 3, l.Insert("a"))
	assert.Equal(t, 4, l.Insert("b"))
	assert.Equal(t, 5, l.Insert("c"))

	bs := l.filterBS(func(item *string) bool {
		return *item == "a"
	})

	assert.Equal(t, []uint32{0, 3}, bs.ToSlice())
}
