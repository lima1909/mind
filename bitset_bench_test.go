package mind

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkBitSetContains(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		assert.True(b, bs.Contains(found_val))
	}
}

func BenchmarkBitSetCount(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		assert.Equal(b, count, bs.Count())
	}
}

func BenchmarkBitSetAnd(b *testing.B) {
	bs1 := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%3 == 0 {
			bs1.Set(uint32(i))
		}
	}
	bs2 := NewBitSet[uint32]()
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

func BenchmarkBitSetOr(b *testing.B) {
	bs1 := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%3 == 0 {
			bs1.Set(uint32(i))
		}
	}
	bs2 := NewBitSet[uint32]()
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

func BenchmarkBitSetXor(b *testing.B) {
	bs1 := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%3 == 0 {
			bs1.Set(uint32(i))
		}
	}
	bs2 := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		if i%6 == 0 {
			bs2.Set(uint32(i))
		}
	}
	b.ResetTimer()

	for b.Loop() {
		r := bs2.Copy()
		r.Xor(bs1)
		assert.Equal(b, 500_000, r.Count())
	}
}

func BenchmarkBitSetToSlice(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		assert.Equal(b, count, len(bs.ToSlice()))
	}
}

func BenchmarkBitSetValuesIter(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		c := 0
		bs.Values(func(v uint32) bool {
			_ = v
			c += 1
			return true
		})
		assert.Equal(b, count, c)

	}
}

func BenchmarkBitSetValuesBatchIter(b *testing.B) {
	bs := NewBitSet[uint32]()
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

func BenchmarkBitSetShrink(b *testing.B) {
	bs := NewBitSetWithCapacity[uint32](2000)
	bs.Set(1)
	bs.Set(10)
	b.ResetTimer()

	for b.Loop() {
		bs.Shrink()
		assert.Equal(b, 0, bs.MaxSetIndex())

	}
}

func BenchmarkBitSetCopy(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		bsCopy := bs.Copy()
		assert.Equal(b, bsCopy, bs)
	}
}

func BenchmarkBitSetCopyInto(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	buf := make([]uint64, 0, len(bs.data))
	for b.Loop() {
		bsCopy := bs.CopyInto(buf)
		assert.Equal(b, bsCopy, bs)
	}
}

func BenchmarkBitSetCreateNew(b *testing.B) {
	for b.Loop() {
		// 46875
		bs := NewBitSetWithCapacity[uint32](47000)
		for i := 1; i <= count; i++ {
			bs.Set(uint32(i))
		}
	}
}

func BenchmarkBitSetMaxSetIndex(b *testing.B) {
	bs := NewBitSet[uint32]()
	for i := 1; i <= count; i++ {
		bs.Set(uint32(i))
	}
	b.ResetTimer()

	for b.Loop() {
		assert.Equal(b, 46875, bs.MaxSetIndex())
	}
}
