package mind

import (
	"slices"
)

type SliceSet[V Value] struct {
	data []V
}

// NewSliceSet creates a new SliceSet
func NewSliceSet[V Value]() *SliceSet[V] {
	return &SliceSet[V]{data: make([]V, 0)}
}

// NewSliceSetWithCapacity creates a new SliceSet with starting capacity
func NewSliceSetWithCapacity[V Value](size int) *SliceSet[V] {
	return &SliceSet[V]{data: make([]V, 0, size)}
}

// NewSliceSetFrom creates a new SliceSet from given values
func NewSliceSetFrom[V Value](values ...V) *SliceSet[V] {
	s := NewSliceSetWithCapacity[V](len(values))
	for _, v := range values {
		s.Set(v)
	}
	return s
}

// Set inserts or updates the key in the Set
func (s *SliceSet[V]) Set(value V) {
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
func (s *SliceSet[V]) UnSet(value V) bool {
	idx, found := slices.BinarySearch(s.data, value)
	if !found {
		return false
	}

	s.data = append(s.data[:idx], s.data[idx+1:]...)
	return true

}

// Contains check, is the value saved in the Set
func (s *SliceSet[V]) Contains(value V) bool {
	_, found := slices.BinarySearch(s.data, value)
	return found
}

// Min return the min value of this Set
// [1, 3, 100] => 1
// if the set is empty, return -1
func (s *SliceSet[V]) Min() int {
	if len(s.data) == 0 {
		return -1
	}

	return int(s.data[0])
}

// Max return the max value of this Set
// [1, 3, 100] => 100
// if the set is empty, return -1
func (s *SliceSet[V]) Max() int {
	l := len(s.data)
	if l == 0 {
		return -1
	}

	return int(s.data[l-1])
}

// MaxSetIndex return the max index the Set
func (s *SliceSet[V]) MaxSetIndex() int {
	l := len(s.data)
	if l == 0 {
		return -1
	}

	return int(l - 1)
}

// Counts how many values are in the Set, the len of the Set.
func (s *SliceSet[V]) Count() int { return len(s.data) }

// Len how many values are in the Set, the len of the Set.
func (s *SliceSet[V]) Len() int { return len(s.data) }

// Copy copy the complete Set.
func (s *SliceSet[V]) Copy() *SliceSet[V] {
	target := make([]V, len(s.data))
	copy(target, s.data)
	return &SliceSet[V]{data: target}
}

// And computes the logical And, (intersection) of two sorted Set.
func (s *SliceSet[V]) And(other *SliceSet[V]) {
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
func (s *SliceSet[V]) Or(other *SliceSet[V]) {
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
	res := make([]V, 0, la+lo)

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
func (s *SliceSet[V]) Xor(other *SliceSet[V]) {
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
	res := make([]V, 0, la+lo)

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
func (s *SliceSet[V]) AndNot(other *SliceSet[V]) {
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

func (s *SliceSet[V]) Values(yield func(V) bool) {
	sa := s.data
	for _, v := range sa {
		if !yield(v) {
			return
		}
	}
}

func (s *SliceSet[V]) ToSlice() []V         { return s.data }
func (s *SliceSet[V]) ToBitSet() *BitSet[V] { return NewBitSetFrom(s.data...) }
