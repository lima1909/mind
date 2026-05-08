package mind

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrigram_Base(t *testing.T) {
	ti := NewTrigramIndexFrom("apple", "apply", "ban", "banana", "xapp")
	assert.Equal(t, 5, ti.Len())
	assert.Equal(t, []uint32{0, 1, 4}, ti.Get("app").ToSlice())
	assert.Equal(t, []uint32{2, 3}, ti.Get("an").ToSlice())
	// not found
	assert.Equal(t, []uint32{}, ti.Get("nix").ToSlice())

	assert.True(t, ti.Delete(2))
	assert.Equal(t, 4, ti.Len())
	// search with 2 letters
	assert.Equal(t, []uint32{3}, ti.Get("an").ToSlice())

	// grow the bucket list
	ti.Put("xban", 20)
	assert.Equal(t, 5, ti.Len())
	assert.Equal(t, []uint32{3, 20}, ti.Get("ban").ToSlice())

	// reuse index 2
	ti.Put("xappx", 2)
	assert.Equal(t, 6, ti.Len())
	assert.Equal(t, []uint32{0, 1, 2, 4}, ti.Get("app").ToSlice())

	// checks the false positive: ABCD and BCDE matching {0, 2}
	ti = NewTrigramIndexFrom("ABCD", "ZZZ", "BCDE")
	assert.Equal(t, []uint32{}, ti.Get("ABCDE").ToSlice())

	// empty init
	ti = NewTrigramIndex()
	assert.Equal(t, 0, ti.Len())
	assert.Equal(t, []uint32{}, ti.Get("nix").ToSlice())

	ti.Put("üöß€ä@", 2)
	assert.Equal(t, 1, ti.Len())
	assert.Equal(t, []uint32{2}, ti.Get("öß€ä").ToSlice())
}

func TestTrigram_abc(t *testing.T) {
	ti := NewTrigramIndex()
	ti.Put("abc---bcd", 0)
	assert.Equal(t, 1, ti.Len())

	r := ti.Get("abcd")
	assert.Equal(t, 0, r.Count())

	r = ti.Get("abc")
	assert.Equal(t, 1, r.Count())
	r = ti.Get("bcd")
	assert.Equal(t, 1, r.Count())

	ti.Put("abc---abc", 1)
	assert.Equal(t, 2, ti.Len())

	assert.Equal(t, []uint32{0, 1}, ti.Get("abc").ToSlice())
	assert.Equal(t, []uint32{1}, ti.Get("--ab").ToSlice())
}

func TestTrigram_BulkPut(t *testing.T) {
	apple := "apple"
	apply := "apply"
	ban := "ban"
	banana := "banana"
	xapp := "xapp"
	data := slices.All([]*string{&apple, &apply, &ban, &banana, &xapp})
	ti := NewTrigramIndex()
	TrigramIndexBulkPut(&ti, func(s *string) string { return *s }, data)

	assert.Equal(t, 5, ti.Len())
	assert.Equal(t, []uint32{0, 1, 4}, ti.Get("app").ToSlice())
	assert.Equal(t, []uint32{2, 3}, ti.Get("an").ToSlice())
	// not found
	assert.Equal(t, []uint32{}, ti.Get("nix").ToSlice())
}
