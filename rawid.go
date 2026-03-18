package mind

const (
	// sparseMinCount is the minimum element count before considering a switch to BitSet.
	sparseMinCount = 64
)

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
	count := s.Count()
	maxVal := s.Max()

	if maxVal < 0 {
		// empty set — prefer SliceSet
		if s.bits != nil {
			s.slice = NewSliceSet[U]()
			s.bits = nil
		}
		return
	}

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

// Rebalance forces a representation check and switches if beneficial.
// Useful after a sequence of UnSet calls on a BitSet representation.
func (s *RawIDs[U]) Rebalance() { s.rebalance() }

// toBitSet returns the internal BitSet, or creates a temporary one from the SliceSet.
// The returned value must NOT be mutated if it's a temporary.
func (s *RawIDs[U]) toBitSet() *BitSet[U] {
	if s.bits != nil {
		return s.bits
	}
	return s.slice.ToBitSet()
}

// ensureBitSet converts the internal representation to BitSet in-place.
func (s *RawIDs[U]) ensureBitSet() {
	if s.bits == nil {
		s.bits = s.slice.ToBitSet()
		s.slice = nil
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

// Count returns the number of elements in the set.
func (s *RawIDs[U]) Count() int {
	if s.IsSlice() {
		return s.slice.Count()
	}
	return s.bits.Count()
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
	// If both are slices, use the efficient merge-based SliceSet.And
	if s.IsSlice() && other.IsSlice() {
		s.slice.And(other.slice)
		return
	}

	s.ensureBitSet()
	otherBits := other.toBitSet()
	s.bits.And(otherBits)
	s.rebalance()
}

// Or computes the union of two sets.
// The result is stored in the receiver.
func (s *RawIDs[U]) Or(other *RawIDs[U]) {
	if s.IsSlice() && other.IsSlice() {
		s.slice.Or(other.slice)
		s.rebalance()
		return
	}

	s.ensureBitSet()
	otherBits := other.toBitSet()
	s.bits.Or(otherBits)
	s.rebalance()
}

// Xor computes the symmetric difference of two sets.
// The result is stored in the receiver.
func (s *RawIDs[U]) Xor(other *RawIDs[U]) {
	if s.IsSlice() && other.IsSlice() {
		s.slice.Xor(other.slice)
		s.rebalance()
		return
	}

	s.ensureBitSet()
	otherBits := other.toBitSet()
	s.bits.Xor(otherBits)
	s.rebalance()
}

// AndNot removes all elements from the current set that exist in the other set.
// Known as "Difference".
//
// Example: [1, 2, 110, 2345] AndNot [2, 110] => [1, 2345]
func (s *RawIDs[U]) AndNot(other *RawIDs[U]) {
	if s.IsSlice() && other.IsSlice() {
		s.slice.AndNot(other.slice)
		return
	}

	s.ensureBitSet()
	otherBits := other.toBitSet()
	s.bits.AndNot(otherBits)
	s.rebalance()
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
