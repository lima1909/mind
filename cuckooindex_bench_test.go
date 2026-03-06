package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
