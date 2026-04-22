package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitList_Base(t *testing.T) {
	sl := NewSkipList[int, string]()
	assert.Equal(t, 0, sl.Len())

	assert.True(t, sl.Put(1, "a"))
	assert.True(t, sl.Put(3, "c"))
	assert.True(t, sl.Put(2, "b"))
	assert.False(t, sl.Put(2, "b"))

	val, found := sl.Get(2)
	assert.True(t, found)
	assert.Equal(t, "b", val)
	assert.Equal(t, 3, sl.Len())

	assert.True(t, sl.Delete(2))
	val, found = sl.Get(2)
	assert.False(t, found)
	assert.Equal(t, "", val)
	assert.Equal(t, 2, sl.Len())

	assert.False(t, sl.Delete(2))
	assert.Equal(t, 2, sl.Len())

	val, found = sl.Get(1)
	assert.True(t, found)
	assert.Equal(t, "a", val)

	val, found = sl.Get(3)
	assert.True(t, found)
	assert.Equal(t, "c", val)
}

func TestSplitList_NilValue(t *testing.T) {
	sl := NewSkipList[string, *string]()
	sl.Put("a", nil)
	assert.Equal(t, 1, sl.Len())

	val, found := sl.Get("a")
	assert.True(t, found)
	assert.Nil(t, val)
}

func TestSplitList_PutWithZeroValueKey(t *testing.T) {
	sl1 := NewSkipList[string, string]()
	sl1.Put("", "---")
	assert.Equal(t, 1, sl1.Len())
	result1, found1 := sl1.Get("")
	assert.True(t, found1)
	assert.Equal(t, "---", result1)

	sl2 := NewSkipList[int, string]()
	sl2.Put(0, "---")
	sl2.Put(1, "+++")
	result2, found2 := sl2.Get(0)
	assert.True(t, found2)
	assert.Equal(t, "---", result2)
	assert.Equal(t, 2, sl2.Len())
}

func TestSplitList_DeleteAndGetTheZeroValueKey(t *testing.T) {
	sl := NewSkipList[string, string]()
	assert.False(t, sl.Delete(""))
	assert.Equal(t, 0, sl.Len())

	val, found := sl.Get("")
	assert.False(t, found)
	assert.Equal(t, "", val)
}

func TestSplitList_Traverse(t *testing.T) {
	count := 10

	sl := NewSkipList[uint32, uint32]()
	for i := 1; i <= count; i++ {
		sl.Put(uint32(i), uint32(i))
	}
	assert.Equal(t, count, sl.Len())

	c := 0
	toTheEnd := sl.Traverse(func(key, val uint32) bool {
		c += 1
		return true
	})
	assert.True(t, toTheEnd)
	assert.Equal(t, count, c)

	c = 0
	toTheEnd = sl.Traverse(func(key, val uint32) bool {
		c += 1
		return c != 5
	})
	assert.False(t, toTheEnd)
	assert.Equal(t, 5, c)

}

func TestSplitList_Range(t *testing.T) {
	sl := NewSkipList[byte, uint32]()
	sl.Put(1, 1)
	sl.Put(3, 3)
	sl.Put(5, 5)
	sl.Put(4, 4)

	result := make([]uint32, 0)
	sl.Range(2, 42,
		func(key byte, val uint32) bool {
			result = append(result, val)
			return true
		})
	assert.Equal(t, []uint32{3, 4, 5}, result)
}

func TestSplitList_RangeInclusiveTo(t *testing.T) {
	sl := NewSkipList[string, uint32]()
	sl.Put("a", 1)
	sl.Put("c", 3)
	sl.Put("z", 5)
	sl.Put("x", 4)

	result := make([]uint32, 0)
	sl.Range("b", "z",
		func(key string, val uint32) bool {
			result = append(result, val)
			return true
		})
	assert.Equal(t, []uint32{3, 4, 5}, result)
}

func TestSplitList_RangeInclusiveFromTo(t *testing.T) {
	sl := NewSkipList[int, uint32]()
	sl.Put(2, 2)
	sl.Put(3, 3)
	sl.Put(5, 5)
	sl.Put(4, 4)

	result := make([]uint32, 0)
	sl.Range(2, 5,
		func(key int, val uint32) bool {
			result = append(result, val)
			return true
		})
	assert.Equal(t, []uint32{2, 3, 4, 5}, result)
}

func TestSplitList_NotInRange(t *testing.T) {
	sl := NewSkipList[uint32, uint32]()
	sl.Put(1, 1)
	sl.Put(3, 3)

	result := make([]uint32, 0)
	sl.Range(4, 42,
		func(key, val uint32) bool {
			result = append(result, val)
			return true
		})
	assert.Equal(t, 0, len(result))
}

func TestSplitList_FirstValue(t *testing.T) {
	sl := NewSkipList[uint32, uint32]()
	val, ok := sl.FirstValue()
	assert.False(t, ok)
	assert.Equal(t, uint32(0), val)

	sl.Put(1, 1)
	val, ok = sl.FirstValue()
	assert.True(t, ok)
	assert.Equal(t, uint32(1), val)
}

func TestSplitList_LastValue(t *testing.T) {
	sl := NewSkipList[uint32, uint32]()
	val, ok := sl.LastValue()
	assert.False(t, ok)
	assert.Equal(t, uint32(0), val)

	sl.Put(1, 1)
	val, ok = sl.LastValue()
	assert.True(t, ok)
	assert.Equal(t, uint32(1), val)

	sl.Put(5, 5)
	val, ok = sl.LastValue()
	assert.True(t, ok)
	assert.Equal(t, uint32(5), val)
}

func TestSplitList_MinKey(t *testing.T) {
	sl := NewSkipList[int, uint32]()
	sl.Put(1, 2)
	sl.Put(3, 4)

	k, found := sl.MinKey()
	assert.True(t, found)
	assert.Equal(t, 1, k)

	sl.Delete(1)
	k, found = sl.MinKey()
	assert.True(t, found)
	assert.Equal(t, 3, k)

	sl.Delete(3)
	k, found = sl.MinKey()
	assert.False(t, found)
	assert.Equal(t, 0, k)
}

func TestSplitList_MaxKey(t *testing.T) {
	sl := NewSkipList[int, uint32]()
	sl.Put(1, 2)
	sl.Put(3, 4)

	k, found := sl.MaxKey()
	assert.True(t, found)
	assert.Equal(t, 3, k)

	sl.Delete(3)
	k, found = sl.MaxKey()
	assert.True(t, found)
	assert.Equal(t, 1, k)

	sl.Delete(1)
	k, found = sl.MaxKey()
	assert.False(t, found)
	assert.Equal(t, 0, k)
}

func TestSplitList_Less(t *testing.T) {
	sl := NewSkipList[int, string]()
	assert.True(t, sl.Put(1, "a"))
	assert.True(t, sl.Put(3, "c"))
	assert.True(t, sl.Put(2, "b"))
	assert.True(t, sl.Put(5, "b"))

	result := make([]int, 0)
	sl.Less(1, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{}, result)

	result = result[:0]
	sl.Less(3, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{1, 2}, result)

	result = result[:0]
	sl.Less(99, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 5}, result)
}

func TestSplitList_LessEqual(t *testing.T) {
	sl := NewSkipList[int, string]()
	assert.True(t, sl.Put(1, "a"))
	assert.True(t, sl.Put(3, "c"))
	assert.True(t, sl.Put(2, "b"))
	assert.True(t, sl.Put(5, "b"))

	result := make([]int, 0)
	sl.LessEqual(0, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{}, result)

	result = result[:0]
	sl.LessEqual(3, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{1, 2, 3}, result)

	result = result[:0]
	sl.LessEqual(99, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 5}, result)
}

func TestSplitList_Greater(t *testing.T) {
	sl := NewSkipList[int, string]()
	assert.True(t, sl.Put(1, "a"))
	assert.True(t, sl.Put(3, "c"))
	assert.True(t, sl.Put(2, "b"))
	assert.True(t, sl.Put(5, "b"))

	result := make([]int, 0)
	sl.Greater(0, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 5}, result)

	result = result[:0]
	sl.Greater(3, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{5}, result)

	result = result[:0]
	sl.Greater(99, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{}, result)
}

func TestSplitList_GreaterEqual(t *testing.T) {
	sl := NewSkipList[int, string]()
	assert.True(t, sl.Put(1, "a"))
	assert.True(t, sl.Put(3, "c"))
	assert.True(t, sl.Put(2, "b"))
	assert.True(t, sl.Put(5, "b"))

	result := make([]int, 0)
	sl.GreaterEqual(0, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 5}, result)

	result = result[:0]
	sl.GreaterEqual(3, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{3, 5}, result)

	result = result[:0]
	sl.GreaterEqual(99, func(key int, val string) bool {
		result = append(result, key)
		return true
	})
	assert.Equal(t, []int{}, result)
}

func TestSplitList_StringStartsWith(t *testing.T) {
	sl := NewSkipList[string, int]()
	assert.True(t, sl.Put("abc", 1))
	assert.True(t, sl.Put("bca", 2))
	assert.True(t, sl.Put("bcx", 3))
	assert.True(t, sl.Put("ba", 4))

	var result []int
	sl.StringStartsWith("bc", func(k string, v int) bool {
		result = append(result, v)
		return true
	})

	assert.Equal(t, []int{2, 3}, result)
}

func TestSplitList_FindSortedKeys(t *testing.T) {
	sl := NewSkipList[byte, uint32]()
	sl.Put(1, 1)
	sl.Put(3, 3)
	sl.Put(5, 5)
	sl.Put(4, 4)

	// sorted
	counVisit := 0
	result := make([]uint32, 0)
	sl.FindFromSortedKeys(func(key byte, val uint32) bool {
		result = append(result, val)
		// ignore 0 and 7
		counVisit++
		return true
	}, 0, 1, 5, 7)
	assert.Equal(t, []uint32{1, 5}, result)
	assert.Equal(t, 2, counVisit)

	// NOT sorted
	counVisit = 0
	result = make([]uint32, 0)
	sl.FindFromSortedKeys(func(key byte, val uint32) bool {
		result = append(result, val)
		// ignore 0 and 7
		counVisit++
		return true
	}, 0, 5, 7, 1)
	assert.Equal(t, []uint32{5}, result)
	assert.Equal(t, 1, counVisit)
}
