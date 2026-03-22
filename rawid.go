package mind

const (
	// sparseMinCount is the minimum element count before considering a switch to BitSet.
	sparseMinCount = 64
)

// RawIDs32 is the default RawIDs
type RawIDs32 = RawIDs[uint32]

// RawIDs saved the internal position (index) from a list (slice).
// It is a hybrid set that automatically switches between a SliceSet
// (for sparse or small data) and a BitSet (for dense data) based on actual
// data characteristics. This gives optimal performance across different
// data distributions:
//
//   - SliceSet: O(log n) lookup, O(n) memory per element. Best for few elements
//     spread over a large range (sparse data).
//   - BitSet: O(1) lookup, O(max/64) memory. Best for many elements in a
//     compact range (dense data).
//
// The set starts as a SliceSet and promotes to BitSet when the element count
// grows large enough and the data is dense. It demotes back after set operations
// that reduce count (And, AndNot) when the result becomes sparse.
type RawIDs[U UInt] struct {
	slice *SliceSet[U]
	bits  *BitSet[U]
}

// NewRawID creates a new empty Set, starting with a SliceSet representation.
func NewRawIDs[U UInt]() *RawIDs[U] {
	return &RawIDs[U]{slice: NewSliceSet[U]()}
}

// NewRawIDWithCapacity creates a new Set with starting capacity.
func NewRawIDsWithCapacity[U UInt](size int) *RawIDs[U] {
	return &RawIDs[U]{slice: NewSliceSetWithCapacity[U](size)}
}

// NewRawIDFrom creates a new Set from given values.
// Automatically selects the best representation based on the data.
func NewRawIDsFrom[U UInt](values ...U) *RawIDs[U] {
	s := &RawIDs[U]{slice: NewSliceSetFrom(values...)}
	s.rebalance()
	return s
}

// IsSlice returns true if the current representation is a SliceSet.
func (s *RawIDs[U]) IsSlice() bool { return s.slice != nil }

// IsBitSet returns true if the current internal representation is a BitSet.
func (s *RawIDs[U]) IsBitSet() bool { return s.bits != nil }

// shouldPromote decides if the data justifies a BitSet representation.
// BitSet is preferred when:
//  1. There are enough elements (above sparseMinCount)
//  2. The data is dense enough that a BitSet uses comparable or less memory.
//
// Memory comparison:
//   - BitSet:    (max/64 + 1) * 8 bytes
//   - SliceSet:  count * 8 bytes  (for uint64-sized elements, less for smaller types)
//
// We give BitSet a 2x allowance since its O(1) operations make up for the extra memory.
func shouldPromote(count, maxUInt int) bool {
	if count <= sparseMinCount {
		return false
	}
	bitsetWords := maxUInt/64 + 1
	return bitsetWords <= count*2
}

// rebalance checks whether the current representation is optimal and switches if needed.
func (s *RawIDs[U]) rebalance() {
	maxVal := s.Max()

	if maxVal < 0 {
		// empty set — prefer SliceSet
		if s.bits != nil {
			s.slice = NewSliceSet[U]()
			s.bits = nil
		}
		return
	}

	count := s.Count()
	promote := shouldPromote(count, maxVal)

	if promote && s.slice != nil {
		// promote: SliceSet → BitSet
		s.bits = s.slice.ToBitSet()
		s.slice = nil
	} else if !promote && s.bits != nil {
		// demote: BitSet → SliceSet
		s.slice = NewSliceSetFrom(s.bits.ToSlice()...)
		s.bits = nil
	}
}

// Set inserts the value into the set.
func (s *RawIDs[U]) Set(value U) {
	if s.IsSlice() {
		s.slice.Set(value)
		// check promotion only when crossing the threshold
		if s.slice.Count() > sparseMinCount {
			s.rebalance()
		}
	} else {
		s.bits.Set(value)
	}
}

// UnSet removes the value from the set.
func (s *RawIDs[U]) UnSet(value U) bool {
	if s.IsSlice() {
		return s.slice.UnSet(value)
	}
	return s.bits.UnSet(value)
}

// Range iterates over set bits between 'from' and 'to' (inclusive).
// It calls 'visit' for each found bit. If 'visit' returns false, iteration stops.
func (s *RawIDs[U]) Range(from, to U, visit func(v U) bool) {
	if s.IsSlice() {
		s.slice.Range(from, to, visit)
		return
	}
	s.bits.Range(from, to, visit)
}

// Contains checks if the value exists in the set.
func (s *RawIDs[U]) Contains(value U) bool {
	if s.IsSlice() {
		return s.slice.Contains(value)
	}
	return s.bits.Contains(value)
}

// Min returns the minimum value in the set, or -1 if empty.
func (s *RawIDs[U]) Min() int {
	if s.IsSlice() {
		return s.slice.Min()
	}
	return s.bits.Min()
}

// Max returns the maximum value in the set, or -1 if empty.
func (s *RawIDs[U]) Max() int {
	if s.IsSlice() {
		return s.slice.Max()
	}
	return s.bits.Max()
}

// MaxSetIndex returns the max index of the underlying storage.
func (s *RawIDs[U]) MaxSetIndex() int {
	if s.IsSlice() {
		return s.slice.MaxSetIndex()
	}
	return s.bits.MaxSetIndex()
}

// ValueOnIndex returns the Value of the dx-th matched item.
func (s *RawIDs[U]) ValueOnIndex(idx uint32) (uint32, bool) {
	if s.IsSlice() {
		return s.slice.ValueOnIndex(idx)
	}
	return s.bits.ValueOnIndex(idx)
}

// Count returns the number of elements in the set.
func (s *RawIDs[U]) Count() int {
	if s.IsSlice() {
		return s.slice.Count()
	}
	return s.bits.Count()
}

// IsEmpty returns the number of elements in the set is equals 0.
func (s *RawIDs[U]) IsEmpty() bool {
	if s.IsSlice() {
		return len(s.slice.data) == 0
	}
	return s.bits.IsEmpty()
}

// Len returns the length of the underlying storage.
func (s *RawIDs[U]) Len() int {
	if s.IsSlice() {
		return s.slice.Len()
	}
	return s.bits.Len()
}

// Copy creates a deep copy of the set, preserving representation.
func (s *RawIDs[U]) Copy() *RawIDs[U] {
	if s.IsSlice() {
		return &RawIDs[U]{slice: s.slice.Copy()}
	}
	return &RawIDs[U]{bits: s.bits.Copy()}
}

// And computes the intersection of two sets.
// The result is stored in the receiver.
func (s *RawIDs[U]) And(other *RawIDs[U]) {
	switch {
	// both are Slices
	case s.bits == nil && other.bits == nil:
		s.slice.And(other.slice)
	// both are BitSets
	case s.bits != nil && other.bits != nil:
		s.bits.And(other.bits)
	// BitSet and Slice
	case s.bits != nil && other.bits == nil:
		// Intersecting a massive BitSet with a small Slice MUST result in a Slice!
		// We demote ourselves instantly, extracting only the matched bits.
		newSlice := make([]U, 0, len(other.slice.data))
		for _, val := range other.slice.data {
			if s.bits.Contains(val) {
				newSlice = append(newSlice, val)
			}
		}
		s.slice = &SliceSet[U]{data: newSlice}
		s.bits = nil
	// Slice and BitSet
	case s.bits == nil && other.bits != nil:
		var n int
		for _, val := range s.slice.data {
			if other.bits.Contains(val) {
				s.slice.data[n] = val
				n++
			}
		}
		s.slice.data = s.slice.data[:n] // Truncate to new size instantly
	}
}

// Or computes the union of two sets.
// The result is stored in the receiver.
func (s *RawIDs[U]) Or(other *RawIDs[U]) {
	switch {
	// both are Slices
	case s.bits == nil && other.bits == nil:
		s.slice.Or(other.slice)
		if len(s.slice.data) > sparseMinCount {
			s.rebalance()
		}
	// both are BitSets
	case s.bits != nil && other.bits != nil:
		s.bits.Or(other.bits)
	// BitSet and Slice
	case s.bits != nil && other.bits == nil:
		// Just turn on the bits from their slice. Zero allocations!
		for _, val := range other.slice.data {
			s.bits.Set(val)
		}
	// Slice and BitSet
	case s.bits == nil && other.bits != nil:
		// A BitSet dominates a Slice. We clone their BitSet,
		// then insert our slice items into it.
		s.bits = other.bits.Copy()
		for _, val := range s.slice.data {
			s.bits.Set(val)
		}
		s.slice = nil
	}
}

// Xor computes the symmetric difference of two sets.
// The result is stored in the receiver.
func (s *RawIDs[U]) Xor(other *RawIDs[U]) {
	switch {
	// both are Slices
	case s.bits == nil && other.bits == nil:
		s.slice.Xor(other.slice)
		// Symmetric difference can grow larger than the original slice,
		// so we check if promotion is needed.
		if len(s.slice.data) > sparseMinCount {
			s.rebalance()
		}
	// both are BitSets
	case s.bits != nil && other.bits != nil:
		s.bits.Xor(other.bits)
	// BitSet and Slice
	case s.bits != nil && other.bits == nil:
		// Flip the bits from their slice into our bitset.
		// If the bit was 1, it becomes 0. If it was 0, it becomes 1.
		for _, val := range other.slice.data {
			s.bits.flipTheBit(val)
		}
	// Slice and BitSet
	case s.bits == nil && other.bits != nil:
		// A BitSet usually dominates. Copy their BitSet and flip our slice bits into it.
		newBits := other.bits.Copy()
		for _, val := range s.slice.data {
			newBits.flipTheBit(val)
		}
		s.bits = newBits
		s.slice = nil
	}
}

// AndNot removes all elements from the current set that exist in the other set.
// Known as "Difference".
//
// Example: [1, 2, 110, 2345] AndNot [2, 110] => [1, 2345]
func (s *RawIDs[U]) AndNot(other *RawIDs[U]) {
	switch {
	// both are Slices
	case s.bits == nil && other.bits == nil:
		s.slice.AndNot(other.slice)
	// both are BitSets
	case s.bits != nil && other.bits != nil:
		s.bits.AndNot(other.bits)
	// BitSet and Slice
	case s.bits != nil && other.bits == nil:
		// Just unset the specific bits they have in their slice
		for _, val := range other.slice.data {
			s.bits.UnSet(val)
		}
	// Slice and BitSet
	case s.bits == nil && other.bits != nil:
		// Filter our slice IN-PLACE. If they have the bit, we drop the item.
		var n int
		for _, val := range s.slice.data {
			if !other.bits.Contains(val) {
				s.slice.data[n] = val
				n++
			}
		}
		s.slice.data = s.slice.data[:n]
	}
}

// UInts iterates over all values in the set.
func (s *RawIDs[U]) Values(yield func(U) bool) {
	if s.IsSlice() {
		s.slice.Values(yield)
	} else {
		s.bits.Values(yield)
	}
}

// ToSlice returns all values as a sorted slice.
func (s *RawIDs[U]) ToSlice() []U {
	if s.IsSlice() {
		return s.slice.ToSlice()
	}
	return s.bits.ToSlice()
}

// ToBitSet returns a BitSet copy of this set.
func (s *RawIDs[U]) ToBitSet() *BitSet[U] {
	if s.bits != nil {
		return s.bits.Copy()
	}
	return s.slice.ToBitSet()
}

// ToSliceSet returns a SliceSet copy of this set.
func (s *RawIDs[U]) ToSliceSet() *SliceSet[U] {
	if s.slice != nil {
		return s.slice.Copy()
	}
	return NewSliceSetFrom(s.bits.ToSlice()...)
}
