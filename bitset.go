package mind

import (
	"math/bits"
)

type UInt interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// BitSet32 is the default BitSet
type BitSet32 = BitSet[uint32]

type BitSet[U UInt] struct {
	data  []uint64
	count int // cached popcount; -1 means dirty (needs recount)
}

const defaultSize = (1 << 16) / 64

// NewBitSet creates a new BitSet
func NewBitSet[U UInt]() *BitSet[U] {
	return &BitSet[U]{data: make([]uint64, 0, defaultSize)}
}

// NewEmptyBitSet creates a new BitSet with len and cap = 0
func NewEmptyBitSet[U UInt]() *BitSet[U] {
	return &BitSet[U]{data: make([]uint64, 0)}
}

// NewBitSetWithCapacity creates a new BitSet with starting capacity
func NewBitSetWithCapacity[U UInt](bits int) *BitSet[U] {
	words := (bits + 63) >> 6
	return &BitSet[U]{data: make([]uint64, 0, words)}
}

// NewBitSetFrom creates a new BitSet from given values
func NewBitSetFrom[U UInt](values ...U) *BitSet[U] {
	var maxVal U
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}

	b := NewBitSetWithCapacity[U](int(maxVal) + 1)
	for _, v := range values {
		b.Set(v)
	}

	return b
}

//go:inline
func (b *BitSet[U]) grow(targetIndex int) {
	needed := targetIndex + 1 - len(b.data)
	if needed > 0 {
		// runtime optimizes this allocation pattern heavily.
		// it will handle the capacity doubling strategy for us.
		b.data = append(b.data, make([]uint64, needed)...)
	}
}

// Set inserts or updates the key in the BitSet
func (b *BitSet[U]) Set(value U) {
	// i>>6 is equals i/64 but faster
	// i&63 is the same: i%64, but faster

	index := int(value) >> 6
	bit := uint64(1) << (value & 63)

	if index >= len(b.data) {
		b.grow(index)
	}

	old := b.data[index]
	b.data[index] = old | bit
	// if the bit was not already set, increment count
	if old&bit == 0 && b.count >= 0 {
		b.count++
	}
}

// UnSet removes the key from the BitSet. Clear the bit value to 0.
func (b *BitSet[U]) UnSet(value U) bool {
	index := int(value) >> 6
	if index < len(b.data) {
		bit := uint64(1) << (value & 63)
		old := b.data[index]
		b.data[index] = old &^ bit
		// if the bit was set, decrement count
		if old&bit != 0 && b.count >= 0 {
			b.count--
		}
		return true
	}

	return false
}

// Contains check, is the value saved in the BitSet
func (b *BitSet[U]) Contains(value U) bool {
	index := int(value) >> 6
	if index >= len(b.data) {
		return false
	}

	return (b.data[index] & (1 << (value & 63))) != 0
}

// ValueOnIndex returns the Value of the dx-th matched item.
// For exmaple: BitSet Values: [1, 2, 8, 42, 1028]
// 0 -> 1
// 1 -> 2
// 2 -> 8
// 3 -> 42
// 4 -> 1028
// 5 -> not found
func (b *BitSet[U]) ValueOnIndex(idx uint32) (uint32, bool) {
	for i, word := range b.data {
		if word == 0 {
			continue
		}

		// counts how many '1's are in this 64-bit word in a single CPU cycle.
		pop := uint32(bits.OnesCount64(word))
		// if the matches we need are further ahead, skip this ENTIRE block!
		if idx >= pop {
			idx -= pop
			continue
		}

		// the exact bit we want is inside this specific 64-bit word.
		// We use Brian Kernighan's Algorithm to clear the lowest set bit 'k' times.
		for j := uint32(0); j < idx; j++ {
			word &= word - 1 // Magic: Erases the lowest '1' bit
		}

		// Now, the bit we are looking for is the lowest remaining '1'.
		// TrailingZeros64 tells us exactly which bit position it is (0 to 63).
		bitPos := uint32(bits.TrailingZeros64(word))

		// calculate the absolute index in the List
		absoluteIndex := uint32(i*64) + bitPos
		return absoluteIndex, true
	}

	return 0, false
}

// Range iterates over set bits between 'from' and 'to' (inclusive).
// It calls 'visit' for each found bit. If 'visit' returns false, iteration stops.
func (b *BitSet[U]) Range(from, to U, visit func(v U) bool) {
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
			val := U(i<<6) + U(t)

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
func (b *BitSet[U]) Min() int {
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
func (b *BitSet[U]) Max() int {
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
func (b *BitSet[U]) MaxSetIndex() int {
	bl := len(b.data)
	bd := b.data

	for i := bl - 1; i >= 0; i-- {
		if bd[i] != 0 {
			return i
		}
	}

	return -1
}

// Count returns how many bits are set in the BitSet.
// Uses a cached value when available (O(1)), recounts only after bulk operations.
func (b *BitSet[U]) Count() int {
	if b.count >= 0 {
		return b.count
	}

	n := 0
	for _, w := range b.data {
		n += bits.OnesCount64(w)
	}
	b.count = n
	return n
}

// IsEmpty there are no bits set
func (b *BitSet[U]) IsEmpty() bool {
	if b.count == 0 {
		return true
	}
	if b.count > 0 {
		return false
	}
	// count is dirty, check words directly
	for _, w := range b.data {
		if w != 0 {
			return false
		}
	}
	return true
}

// Len returns the len of the bit slice
func (b *BitSet[U]) Len() int { return len(b.data) }

// how many bytes is using
func (b *BitSet[U]) usedBytes() int { return 24 + (len(b.data) * 8) }

// Clear removes all bits
func (b *BitSet[U]) Clear() {
	b.data = b.data[:0]
	b.count = 0
}

// Copy copy the complete BitSet.
func (b *BitSet[U]) Copy() *BitSet[U] {
	target := make([]uint64, len(b.data))
	copy(target, b.data)
	return &BitSet[U]{data: target, count: b.count}
}

// CopyInto copies the current BitSet into the provided buffer.
// It returns a new BitSet wrapper sharing the provided buffer.
// Assumption: cap(buf) >= len(b.data), if not, then panic.
func (b *BitSet[U]) CopyInto(buf []uint64) *BitSet[U] {
	needed := len(b.data)

	if cap(buf) < needed {
		panic("BitSet.CopyInto: buffer too small")
	}

	target := buf[:needed]
	copy(target, b.data)

	return &BitSet[U]{data: target, count: b.count}
}

// And is the logical AND of two BitSet
// In this BitSet is the result, this means the values will be overwritten!
func (b *BitSet[U]) And(other *BitSet[U]) {
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
	b.count = -1 // invalidate cached count
}

// Or is the logical OR of two BitSet
func (b *BitSet[U]) Or(other *BitSet[U]) {
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
	b.count = -1 // invalidate cached count
}

// XOr is the logical XOR of two BitSet
func (b *BitSet[U]) Xor(other *BitSet[U]) {
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
	b.count = -1 // invalidate cached count
}

// AndNot removes all elements from the current set that exist in another set.
// Known as "Bit Clear" or "Set Difference"
//
// Example: [1, 2, 110, 2345] AndNot [2, 110] => [1, 2345]
func (b *BitSet[U]) AndNot(other *BitSet[U]) {
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
	b.count = -1 // invalidate cached count
}

// Shrink trims the bitset to ensure that len(b.data) always points to the last truly useful word.
//
// Operation    Can Grow?    Can Shrink?
// OR            Yes         No
// XOR           Yes         Yes
// AND           No          Yes
// AND NOT       No          Yes
func (b *BitSet[U]) Shrink() {
	bd := b.data

	// start from the end
	i := len(bd) - 1
	for i >= 0 && bd[i] == 0 {
		i--
	}

	b.data = bd[:i+1]
	if i < 0 {
		b.count = 0
	}
}

// Values iterate over the complete BitSet and call the yield function, for every value
func (b *BitSet[U]) Values(yield func(U) bool) {
	for i, w := range b.data {
		for w != 0 {
			t := bits.TrailingZeros64(w)
			val := (i << 6) + t
			if !yield(U(val)) {
				return
			}
			w &= (w - 1)
		}
	}
}

func (b *BitSet[U]) ValuesBatch(yield func([]U) bool) {
	const batchSize = 256
	buffer := make([]U, batchSize)
	pos := 0

	for i, w := range b.data {
		base := i << 6
		for w != 0 {
			t := bits.TrailingZeros64(w)
			buffer[pos] = U(base + t)
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
func (b *BitSet[U]) ToSlice() []U {
	res := make([]U, 0, b.Count())
	b.Values(func(v U) bool {
		res = append(res, v)
		return true
	})
	return res
}
