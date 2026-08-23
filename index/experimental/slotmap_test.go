package experimental

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlotMap_Base(t *testing.T) {
	l := NewSlotMap[string]()
	assert.Equal(t, Handle{0, 0}, l.Insert("a"))
	assert.Equal(t, Handle{1, 0}, l.Insert("b"))
	assert.Equal(t, Handle{2, 0}, l.Insert("c"))
	assert.Equal(t, 3, l.Len())

	val, found := l.Get(Handle{1, 0})
	assert.True(t, found)
	assert.Equal(t, "b", val)

	assert.False(t, l.Remove(Handle{100, 0}))
	assert.Equal(t, 3, l.Len())

	assert.True(t, l.Remove(Handle{1, 0}))
	assert.Equal(t, 2, l.Len())
	_, found = l.Get(Handle{1, 0})
	assert.False(t, found)
	assert.False(t, l.slots[1].occupied)

	// add "z", where "b" was (z replaced b, same Index, different Generation)
	assert.Equal(t, Handle{1, 1}, l.Insert("z"))

	_, found = l.Get(Handle{1, 0})
	assert.False(t, found)

	val, found = l.Get(Handle{1, 1})
	assert.True(t, found)
	assert.Equal(t, "z", val)
}

func TestSlotMap_Update(t *testing.T) {
	l := NewSlotMap[string]()
	assert.Equal(t, Handle{0, 0}, l.Insert("a"))
	assert.Equal(t, Handle{1, 0}, l.Insert("b"))
	assert.Equal(t, Handle{2, 0}, l.Insert("c"))
	assert.Equal(t, 3, l.Len())

	old, ok := l.Update(Handle{1, 0}, "z")
	assert.True(t, ok)
	assert.Equal(t, "b", old)

	// index to big
	_, ok = l.Update(Handle{100, 0}, "z")
	assert.False(t, ok)
	// wrong generation
	_, ok = l.Update(Handle{1, 1}, "z")
	assert.False(t, ok)

	// index not found
	assert.True(t, l.Remove(Handle{1, 0}))
	assert.Equal(t, 2, l.Len())
	_, ok = l.Update(Handle{1, 0}, "z")
	assert.False(t, ok)
}

func TestSlotMap_Iter(t *testing.T) {
	l := NewSlotMap[string]()
	assert.Equal(t, Handle{0, 0}, l.Insert("a"))
	assert.Equal(t, Handle{1, 0}, l.Insert("b"))
	assert.Equal(t, Handle{2, 0}, l.Insert("c"))

	for h, item := range l.Iter() {
		switch h.index {
		case 0:
			assert.Equal(t, "a", item)
		case 1:
			assert.Equal(t, "b", item)
		case 2:
			assert.Equal(t, "c", item)
		default:
			assert.Failf(t, "invalid", "idx: %v", h.index)
		}
	}

	// remove one item in the middle
	assert.True(t, l.Remove(Handle{1, 0}))
	for h, item := range l.Iter() {
		switch h.index {
		case 0:
			assert.Equal(t, "a", item)
		case 2:
			assert.Equal(t, "c", item)
		default:
			assert.Failf(t, "invalid", "idx: %v", h.index)
		}
	}
}

func TestSlotMap_CompactUnstable(t *testing.T) {
	l := NewSlotMap[string]()
	ah := l.Insert("a")
	bh := l.Insert("b")
	ch := l.Insert("c")
	_ = l.Insert("d")
	eh := l.Insert("e")
	_ = l.Insert("f")

	l.Remove(bh) // b
	l.Remove(ch) // c
	l.Remove(eh) // e

	l.Compact(func(oldIndex, newIndex uint32) {
		switch oldIndex {
		// move d from 3 to 1
		case 3:
			assert.Equal(t, uint32(1), newIndex)
		// move f from 3 to 2
		case 5:
			assert.Equal(t, uint32(2), newIndex)
		default:
			assert.Failf(t, "invalid", "idx: %v", oldIndex)
		}
	})
	assert.Equal(t, 3, len(l.slots))

	val, found := l.Get(ah)
	assert.True(t, found)
	assert.Equal(t, "a", val)

	val, found = l.Get(bh)
	assert.True(t, found)
	assert.Equal(t, "d", val)

	val, found = l.Get(ch)
	assert.True(t, found)
	assert.Equal(t, "f", val)
}

func TestSlotMap_GetMany(t *testing.T) {
	l := NewSlotMap[string]()
	assert.Equal(t, Handle{0, 0}, l.Insert("a"))
	assert.Equal(t, Handle{1, 0}, l.Insert("b"))
	assert.Equal(t, Handle{2, 0}, l.Insert("c"))
	assert.Equal(t, Handle{3, 0}, l.Insert("a"))
	assert.Equal(t, Handle{4, 0}, l.Insert("b"))
	assert.Equal(t, Handle{5, 0}, l.Insert("c"))

	result := make([]string, 0, 3)
	l.getMany([]uint32{0, 3}, &result)
	assert.Equal(t, []string{"a", "a"}, result)
	assert.Equal(t, 2, len(result))
}

func TestSlotMap_Filter(t *testing.T) {
	l := NewSlotMap[string]()
	assert.Equal(t, Handle{0, 0}, l.Insert("a"))
	assert.Equal(t, Handle{1, 0}, l.Insert("b"))
	assert.Equal(t, Handle{2, 0}, l.Insert("c"))
	assert.Equal(t, Handle{3, 0}, l.Insert("a"))
	assert.Equal(t, Handle{4, 0}, l.Insert("b"))
	assert.Equal(t, Handle{5, 0}, l.Insert("c"))

	result := make([]string, 0, 3)
	l.filter([]uint32{0, 1, 3, 4}, &result, func(s *string) bool {
		return *s == "a"
	})
	assert.Equal(t, []string{"a", "a"}, result)
	assert.Equal(t, 2, len(result))
}
