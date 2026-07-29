package experimental

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	count     = 3_000_000
	found_val = 990_000
)

func BenchmarkCuckooIndexGet(b *testing.B) {
	ci := newCuckoo()
	for i := 1; i <= count; i++ {
		ci.Put(uint32(i), uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		_, found := ci.Get(uint32(found_val))
		assert.True(b, found)
	}
}
