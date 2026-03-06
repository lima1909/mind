package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	count     = 3_000_000
	found_val = 990_000
	to        = 3000
)

func BenchmarkGet(b *testing.B) {
	sl := NewSkipList[uint32, uint32]()
	for i := 1; i <= count; i++ {
		sl.Put(uint32(i), uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		_, found := sl.Get(uint32(found_val))
		assert.True(b, found)
	}
}

func BenchmarkTraverse(b *testing.B) {
	sl := NewSkipList[uint32, uint32]()
	for i := 1; i <= count; i++ {
		sl.Put(uint32(i), uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		c := 0
		toTheEnd := sl.Traverse(func(key, val uint32) bool {
			c += 1
			return true
		})
		assert.True(b, toTheEnd)
		assert.Equal(b, count, c)

	}
}

func BenchmarkRange(b *testing.B) {
	sl := NewSkipList[uint32, uint32]()
	for i := 1; i <= count; i++ {
		sl.Put(uint32(i), uint32(i))
	}
	b.ResetTimer()

	result := make([]uint32, 0, to)

	for b.Loop() {
		sl.Range(
			uint32(found_val),
			uint32(found_val+to),
			func(key, val uint32) bool {
				result = append(result, val)
				return true
			},
		)

		assert.Equal(b, to, len(result)-1)
		// clear the results
		result = result[:0]
	}
}

type Person struct{ Name string }

func (p *Person) GetName() string { return p.Name }

func BenchmarkFromField(b *testing.B) {
	p := Person{"ItsMe"}
	fromField := ((*Person).GetName)

	for b.Loop() {
		assert.Equal(b, "ItsMe", fromField(&p))
	}
}

func BenchmarkFromName(b *testing.B) {
	p := Person{"ItsMe"}
	fieldName := FromName[Person, string]("Name")

	for b.Loop() {
		assert.Equal(b, "ItsMe", fieldName(&p))
	}
}

func BenchmarkSkiplist_FindSortedKey(b *testing.B) {
	sl := NewSkipList[uint32, uint32]()
	for i := 1; i <= count; i++ {
		sl.Put(uint32(i), uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		count := 0
		sl.FindFromSortedKeys(func(key, val uint32) bool {
			count++
			return true
		}, 1, 10_000, 100_000, 1000_000)
		assert.Equal(b, 4, count)
	}
}

func BenchmarkSkiplist_FindSortedKey99(b *testing.B) {
	sl := NewSkipList[uint32, uint32]()
	for i := 1; i <= count; i++ {
		sl.Put(uint32(i), uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		_, found := sl.Get(1)
		assert.True(b, found)

		_, found = sl.Get(10_000)
		assert.True(b, found)

		_, found = sl.Get(100_000)
		assert.True(b, found)

		_, found = sl.Get(1000_000)
		assert.True(b, found)
	}
}
