package main

import (
	"math/bits"
)

type Value interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type BitSet[V Value] struct {
	data []uint64
}

const defaultSize = (1 << 16) / 64

// NewBitSet creates a new BitSet
func NewBitSet[V Value]() *BitSet[V] {
	return &BitSet[V]{data: make([]uint64, 0, defaultSize)}
}

// NewBitSetWithCapacity creates a new BitSet with starting capacity
func NewBitSetWithCapacity[V Value](bits int) *BitSet[V] {
	words := (bits + 63) >> 6
	return &BitSet[V]{data: make([]uint64, 0, words)}
}

// NewBitSetFrom creates a new BitSet from given values
func NewBitSetFrom[V Value](values ...V) *BitSet[V] {
	var maxVal V
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	b := NewBitSetWithCapacity[V](int(maxVal) + 1)
	for _, v := range values {
		b.Set(v)
	}

	return b
}

//go:inline
func (b *BitSet[V]) grow(targetIndex int) {
	needed := targetIndex + 1 - len(b.data)
	if needed > 0 {
		// runtime optimizes this allocation pattern heavily.
		// it will handle the capacity doubling strategy for us.
		b.data = append(b.data, make([]uint64, needed)...)
	}
}

// Set inserts or updates the key in the BitSet
func (b *BitSet[V]) Set(value V) {
	// i>>6 is equals i/64 but faster
	// i&63 is the same: i%64, but faster

	index := int(value) >> 6
	bit := uint64(1) << (value & 63)

	if index >= len(b.data) {
		b.grow(index)
	}

	b.data[index] |= bit
}

// UnSet removes the key from the BitSet. Clear the bit value to 0.
func (b *BitSet[V]) UnSet(value V) bool {
	index := int(value) >> 6
	if index < len(b.data) {
		b.data[index] &^= (1 << (value & 63))
		return true
	}

	return false
}

// Contains check, is the value saved in the BitSet
func (b *BitSet[V]) Contains(value V) bool {
	index := int(value) >> 6
	if index >= len(b.data) {
		return false
	}

	return (b.data[index] & (1 << (value & 63))) != 0
}

// Range iterates over set bits between 'from' and 'to' (inclusive).
// It calls 'visit' for each found bit. If 'visit' returns false, iteration stops.
func (b *BitSet[V]) Range(from, to V, visit func(v V) bool) {
	if from > to || len(b.data) == 0 {
		return
	}

	startWord := int(from >> 6)
	endWord := int(to >> 6)

	// bounds check
	if startWord >= len(b.data) {
		return
	}
	if endWord >= len(b.data) {
		endWord = len(b.data) - 1
	}

	for i := startWord; i <= endWord; i++ {
		w := b.data[i]
		if w == 0 {
			continue
		}

		if i == startWord {
			w &= (^uint64(0) << (from & 63))
		}
		if i == endWord {
			w &= (^uint64(0) >> (63 - (to & 63)))
		}

		for w != 0 {
			t := bits.TrailingZeros64(w)
			val := V(i<<6) + V(t)

			if !visit(val) {
				return
			}

			w &= w - 1
		}
	}
}

// Min return the min value where an Bit is set
// [1, 3, 100] => 1
// if no max found, return -1
func (b *BitSet[V]) Min() int {
	bd := b.data

	for i, w := range bd {
		if w != 0 {
			// bits.TrailingZeros64 returns the number of zero bits
			// before the first set bit (the "1").
			// Example: w = ...1000 (binary) -> TrailingZeros64 returns 3.
			// The index of that bit is exactly 3.
			return (i << 6) + bits.TrailingZeros64(w)
		}
	}

	return -1
}

// Max return the max value where an Bit is set
// [1, 3, 100] => 100
// if no max found, return -1
func (b *BitSet[V]) Max() int {
	bl := len(b.data)
	bd := b.data

	for i := bl - 1; i >= 0; i-- {
		w := bd[i]
		if w != 0 {
			// bits.Len64 returns the minimum bits to represent w.
			// Example: w = 0...0101 (binary) -> Len64 returns 3.
			// The index of that bit is 3 - 1 = 2.
			return (i << 6) + (bits.Len64(w) - 1)
		}
	}

	return -1
}

// MaxSetIndex return the max index where an Bit is set
func (b *BitSet[V]) MaxSetIndex() int {
	bl := len(b.data)
	bd := b.data

	for i := bl - 1; i >= 0; i-- {
		if bd[i] != 0 {
			return i
		}
	}

	return -1
}

// Counts how many values are in the BitSet, bits are set.
func (b *BitSet[V]) Count() int {
	count := 0
	for _, w := range b.data {
		count += bits.OnesCount64(w)
	}
	return count
}

// IsEmpty there are no bits set, means Count() == 0
func (b *BitSet[V]) IsEmpty() bool { return b.Count() == 0 }

// Len returns the len of the bit slice
func (b *BitSet[V]) Len() int { return len(b.data) }

// how many bytes is using
func (b *BitSet[V]) usedBytes() int { return 24 + (len(b.data) * 8) }

// Clear removes all bits
func (b *BitSet[V]) Clear() { b.data = b.data[:0] }

// Copy copy the complete BitSet.
func (b *BitSet[V]) Copy() *BitSet[V] {
	target := make([]uint64, len(b.data))
	copy(target, b.data)
	return &BitSet[V]{data: target}
}

// CopyInto copies the current BitSet into the provided buffer.
// It returns a new BitSet wrapper sharing the provided buffer.
// Assumption: cap(buf) >= len(b.data), if not, then panic.
func (b *BitSet[V]) CopyInto(buf []uint64) *BitSet[V] {
	needed := len(b.data)

	if cap(buf) < needed {
		panic("BitSet.CopyInto: buffer too small")
	}

	target := buf[:needed]
	copy(target, b.data)

	return &BitSet[V]{data: target}
}

// And is the logical AND of two BitSet
// In this BitSet is the result, this means the values will be overwritten!
func (b *BitSet[V]) And(other *BitSet[V]) {
	l := min(len(b.data), len(other.data))

	// zero out the tail to prevent "Zombie Bits"
	clear(b.data[l:])
	b.data = b.data[:l]

	// BCE: Bounds Check Elimination
	a := b.data
	o := other.data[:l]

	for i := range l {
		a[i] &= o[i]
	}
}

// Or is the logical OR of two BitSet
func (b *BitSet[V]) Or(other *BitSet[V]) {
	od := other.data
	ol := len(od)
	bl := len(b.data)

	if ol == 0 {
		return
	}

	overlap := min(bl, ol)

	// Ensure b.data has enough length for the result
	if bl < ol {
		if cap(b.data) >= ol {
			b.data = b.data[:ol]
		} else {
			b.grow(ol - 1)
		}
		// Copy non-overlapping tail: 0 | x = x
		copy(b.data[overlap:ol], od[overlap:ol])
	}

	// OR the overlapping words
	dst := b.data[:overlap]
	src := od[:overlap]

	for i := range overlap {
		dst[i] |= src[i]
	}
}

// XOr is the logical XOR of two BitSet
func (b *BitSet[V]) Xor(other *BitSet[V]) {
	bl := len(b.data)
	ol := len(other.data)

	overlap := min(bl, ol)
	if overlap == 0 {
		return
	}

	// If 'other' is longer, we simply append its tail to 'b'.
	// Why? Because: 0 (current tail of b) XOR Value (tail of other) = Value.
	if ol > bl {
		b.data = append(b.data, other.data[bl:]...)
	}

	// if 'b' is longer, its tail remains untouched.
	// value (tail of b) XOR 0 (implicit tail of other) = Value.
	bd := b.data
	od := other.data

	_ = bd[overlap-1]
	_ = od[overlap-1]

	for i := range overlap {
		bd[i] ^= od[i]
	}
}

// AndNot removes all elements from the current set that exist in another set.
// Known as "Bit Clear" or "Set Difference"
//
// Example: [1, 2, 110, 2345] AndNot [2, 110] => [1, 2345]
func (b *BitSet[V]) AndNot(other *BitSet[V]) {
	if len(other.data) == 0 || len(b.data) == 0 {
		return
	}

	bd := b.data
	od := other.data
	l := min(len(bd), len(od))

	// eliminates checks inside the loop.
	_ = bd[l-1]
	_ = od[l-1]

	for i := range l {
		bd[i] &^= od[i]
	}
}

// Shrink trims the bitset to ensure that len(b.data) always points to the last truly useful word.
//
// Operation    Can Grow?    Can Shrink?
// OR            Yes         No
// XOR           Yes         Yes
// AND           No          Yes
// AND NOT       No          Yes
func (b *BitSet[V]) Shrink() {
	bd := b.data

	// start from the end
	i := len(bd) - 1
	for i >= 0 && bd[i] == 0 {
		i--
	}

	b.data = bd[:i+1]
}

// Values iterate over the complete BitSet and call the yield function, for every value
func (b *BitSet[V]) Values(yield func(V) bool) {
	for i, w := range b.data {
		for w != 0 {
			t := bits.TrailingZeros64(w)
			val := (i << 6) + t
			if !yield(V(val)) {
				return
			}
			w &= (w - 1)
		}
	}
}

func (b *BitSet[V]) ValuesBatch(yield func([]V) bool) {
	const batchSize = 256
	buffer := make([]V, batchSize)
	pos := 0

	for i, w := range b.data {
		base := i << 6
		for w != 0 {
			t := bits.TrailingZeros64(w)
			buffer[pos] = V(base + t)
			pos++

			// If buffer is full, yield the whole batch
			if pos == batchSize {
				if !yield(buffer) {
					return
				}
				pos = 0
			}
			w &= (w - 1)
		}
	}

	// Yield the final partial batch
	if pos > 0 {
		yield(buffer[:pos])
	}
}

// ToSlice create a new slice which contains all saved values
func (b *BitSet[V]) ToSlice() []V {
	res := make([]V, 0, b.Count())
	b.Values(func(v V) bool {
		res = append(res, v)
		return true
	})
	return res
}
