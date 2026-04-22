package mind

import (
	"slices"
)

type SliceSet[U UInt] struct {
	data []U
}

// NewSliceSet creates a new SliceSet
func NewSliceSet[U UInt]() *SliceSet[U] {
	return &SliceSet[U]{data: make([]U, 0)}
}

// NewSliceSetWithCapacity creates a new SliceSet with starting capacity
func NewSliceSetWithCapacity[U UInt](size int) *SliceSet[U] {
	return &SliceSet[U]{data: make([]U, 0, size)}
}

// NewSliceSetFrom creates a new SliceSet from given values
func NewSliceSetFrom[U UInt](values ...U) *SliceSet[U] {
	s := NewSliceSetWithCapacity[U](len(values))
	for _, v := range values {
		s.Set(v)
	}
	return s
}

// Set inserts or updates the key in the Set
func (s *SliceSet[U]) Set(value U) {
	l := len(s.data)
	if l == 0 {
		s.data = append(s.data, value)
		return
	}
	if value > s.data[l-1] {
		s.data = append(s.data, value)
		return
	}

	idx, found := slices.BinarySearch(s.data, value)
	if found {
		return // already exists
	}

	// grow the slice by one element
	s.data = append(s.data, value)
	// shift elements to the right by 1 to make room at index 'i'
	copy(s.data[idx+1:], s.data[idx:l])
	s.data[idx] = value
}

// UnSet removes the key from the Set.
func (s *SliceSet[U]) UnSet(value U) bool {
	idx, found := slices.BinarySearch(s.data, value)
	if !found {
		return false
	}

	s.data = append(s.data[:idx], s.data[idx+1:]...)
	return true

}

// Range iterates over set bits between 'from' and 'to' (inclusive).
// It calls 'visit' for each found bit. If 'visit' returns false, iteration stops.
func (s *SliceSet[U]) Range(from, to U, visit func(v U) bool) {
	if len(s.data) == 0 || from > to {
		return
	}

	// Binary Search to find the exact starting index.
	// We are looking for the FIRST index where s.data[i] >= from.
	low, high := 0, len(s.data)
	for low < high {
		// Bitwise shift is a micro-optimization for: (low + high) / 2
		mid := int(uint(low+high) >> 1)

		if s.data[mid] < from {
			low = mid + 1
		} else {
			high = mid
		}
	}

	for i := low; i < len(s.data); i++ {
		val := s.data[i]
		if val > to {
			break
		}

		if !visit(val) {
			break
		}
	}
}

// Contains check, is the value saved in the Set
func (s *SliceSet[U]) Contains(value U) bool {
	_, found := slices.BinarySearch(s.data, value)
	return found
}

// ValueOnIndex returns the Value of the dx-th matched item.
func (s *SliceSet[U]) ValueOnIndex(idx uint32) (uint32, bool) {
	if int(idx) >= len(s.data) {
		return 0, false
	}

	return uint32(s.data[idx]), true
}

// Min return the min value of this Set
// [1, 3, 100] => 1
// if the set is empty, return -1
func (s *SliceSet[U]) Min() int {
	if len(s.data) == 0 {
		return -1
	}

	return int(s.data[0])
}

// Max return the max value of this Set
// [1, 3, 100] => 100
// if the set is empty, return -1
func (s *SliceSet[U]) Max() int {
	l := len(s.data)
	if l == 0 {
		return -1
	}

	return int(s.data[l-1])
}

// MaxSetIndex return the max index the Set
func (s *SliceSet[U]) MaxSetIndex() int {
	l := len(s.data)
	if l == 0 {
		return -1
	}

	return int(l - 1)
}

// Counts how many values are in the Set, the len of the Set.
func (s *SliceSet[U]) Count() int { return len(s.data) }

func (s *SliceSet[U]) IsEmpty() bool { return len(s.data) == 0 }

// Len how many values are in the Set, the len of the Set.
func (s *SliceSet[U]) Len() int { return len(s.data) }

// Copy copy the complete Set.
func (s *SliceSet[U]) Copy() *SliceSet[U] {
	target := make([]U, len(s.data))
	copy(target, s.data)
	return &SliceSet[U]{data: target}
}

// And computes the logical And, (intersection) of two sorted Set.
func (s *SliceSet[U]) And(other *SliceSet[U]) {
	la, lo := len(s.data), len(other.data)
	if la == 0 || lo == 0 {
		s.data = s.data[:0]
		return
	}

	if &s.data[0] == &other.data[0] {
		return
	}

	sa, so := s.data, other.data
	i, j, writeIdx := 0, 0, 0

	for i < la && j < lo {
		av := sa[i]
		ov := so[j]

		if av < ov {
			i++
		} else if ov < av {
			j++
		} else {
			sa[writeIdx] = av
			writeIdx++
			i++
			j++
		}
	}

	s.data = sa[:writeIdx]
}

// Or computes the logical OR (union) of two  sorted Set.
func (s *SliceSet[U]) Or(other *SliceSet[U]) {
	la, lo := len(s.data), len(other.data)
	if lo == 0 {
		return
	}
	if la == 0 {
		s.data = other.data
		return
	}

	sa, so := s.data, other.data
	i, j := 0, 0
	res := make([]U, 0, la+lo)

	for i < la && j < lo {
		av, ov := sa[i], so[j]
		if av < ov {
			res = append(res, av)
			i++
		} else if ov < av {
			res = append(res, ov)
			j++
		} else {
			res = append(res, av)
			i++
			j++
		}
	}

	if i < la {
		res = append(res, sa[i:]...)
	} else if j < lo {
		res = append(res, so[j:]...)
	}

	s.data = res
}

// Xor computes the logical XOR  of two  sorted Set.
func (s *SliceSet[U]) Xor(other *SliceSet[U]) {
	la, lo := len(s.data), len(other.data)
	// A XOR 0 = A
	if lo == 0 {
		return
	}
	// 0 XOR B = B
	if la == 0 {
		s.data = append(s.data, other.data...)
		return
	}

	sa, so := s.data, other.data
	i, j := 0, 0
	res := make([]U, 0, la+lo)

	for i < la && j < lo {
		av, ov := sa[i], so[j]
		if av < ov {
			res = append(res, av)
			i++
		} else if ov < av {
			res = append(res, ov)
			j++
		} else {
			i++
			j++
		}
	}

	if i < la {
		res = append(res, sa[i:]...)
	} else if j < lo {
		res = append(res, so[j:]...)
	}

	s.data = res
}

// AndNot removes all elements from the current Set that exist in another Set.
// Known as "Clear" or "Difference"
//
// Example: [1, 2, 110, 2345] AndNot [2, 110] => [1, 2345]
func (s *SliceSet[U]) AndNot(other *SliceSet[U]) {
	la, lo := len(s.data), len(other.data)
	if la == 0 || lo == 0 {
		return
	}

	sa, so := s.data, other.data
	i, j, writeIdx := 0, 0, 0

	for i < la && j < lo {
		av, ov := sa[i], so[j]

		if av < ov {
			// av is not in other, so keep it.
			if writeIdx != i {
				sa[writeIdx] = av
			}
			writeIdx++
			i++
		} else if av > ov {
			j++
		} else {
			i++
			j++
		}
	}

	if i < la {
		if writeIdx != i {
			copy(sa[writeIdx:], sa[i:])
		}
		writeIdx += (la - i)
	}

	s.data = sa[:writeIdx]
}

func (s *SliceSet[U]) Values(yield func(U) bool) {
	sa := s.data
	for _, v := range sa {
		if !yield(v) {
			return
		}
	}
}

func (s *SliceSet[U]) ToSlice() []U         { return s.data }
func (s *SliceSet[U]) ToBitSet() *BitSet[U] { return NewBitSetFrom(s.data...) }
