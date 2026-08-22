package lidx

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

func (s *SliceSet[U]) Values(yield func(U) bool) {
	for _, v := range s.data {
		if !yield(v) {
			return
		}
	}
}

func (s *SliceSet[U]) ValuesBetween(n, m int) []U {
	total := len(s.data)

	if n >= total || n < 0 || total == 0 || m <= 0 {
		return nil
	}

	size := min(m, total-n) + n
	return s.data[n:size]
}

func (s *SliceSet[U]) ValuesBatch(yield func([]U) bool) {
	if len(s.data) == 0 {
		return
	}

	const batchSize = 256
	l := len(s.data)

	batch := l / batchSize
	rest := l % batchSize

	for i := range batch {
		if !yield(s.data[i*batchSize : i*batchSize+batchSize]) {
			return
		}
	}

	yield(s.data[len(s.data)-rest:])
}

// Contains check, is the value saved in the Set
func (s *SliceSet[U]) Contains(value U) bool {
	_, found := slices.BinarySearch(s.data, value)
	return found
}

// ValueOnIndex returns the Value of the dx-th matched item.
// For exmaple: BitSet Values: [1, 2, 8, 42, 1028]
// 0 -> 1
// 1 -> 2
// 2 -> 8
// 3 -> 42
// 4 -> 1028
// 5 -> not found
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

// MaxIndex return the max index the Set
func (s *SliceSet[U]) MaxIndex() int {
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
	if len(s.data) == 0 || len(other.data) == 0 {
		clear(s.data) // Wichtig: Zombie-Werte nullen!
		s.data = s.data[:0]
		return
	}

	if &s.data[0] == &other.data[0] {
		return
	}

	sa := s.data
	so := other.data
	i, j, writeIdx := 0, 0, 0

	for i < len(sa) && j < len(so) {
		av := sa[i]
		ov := so[j]

		if av < ov {
			i++
		} else if av > ov {
			j++
		} else {
			sa[writeIdx] = av
			writeIdx++
			i++
			j++
		}
	}

	clear(sa[writeIdx:])
	s.data = sa[:writeIdx]
}

// Or computes the logical OR (union) of two  sorted Set.
func (s *SliceSet[U]) Or(other *SliceSet[U]) {
	if len(other.data) == 0 {
		return
	}
	if len(s.data) == 0 {
		s.data = append(s.data[:0], other.data...)
		return
	}

	sa := s.data
	so := other.data

	res := make([]U, len(sa)+len(so))
	i, j, k := 0, 0, 0

	for i < len(sa) && j < len(so) {
		av, ov := sa[i], so[j]
		if av < ov {
			res[k] = av
			i++
		} else if av > ov {
			res[k] = ov
			j++
		} else {
			res[k] = av
			i++
			j++
		}
		k++
	}

	if i < len(sa) {
		k += copy(res[k:], sa[i:])
	} else if j < len(so) {
		k += copy(res[k:], so[j:])
	}

	s.data = res[:k]
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
	if len(s.data) == 0 || len(other.data) == 0 {
		return
	}

	sa := s.data
	so := other.data
	i, j, writeIdx := 0, 0, 0

	for i < len(sa) && j < len(so) {
		av, ov := sa[i], so[j]

		if av < ov {
			sa[writeIdx] = av
			writeIdx++
			i++
		} else if av > ov {
			j++
		} else {
			i++
			j++
		}
	}

	if i < len(sa) {
		writeIdx += copy(sa[writeIdx:], sa[i:])
	}

	clear(sa[writeIdx:])
	s.data = sa[:writeIdx]
}

// Removes iterate over the complete SliceStet and call the remove function,
// If remove returns true, the Value will be UnSet
//
// Single in-place compaction pass: O(n) and allocation free. Writing the kept
// values forward as we go is safe even though we read and write the same slice,
// because the write index never runs ahead of the read index.
func (s *SliceSet[U]) Removes(remove func(U) bool) {
	keep := 0

	for _, v := range s.data {
		if remove(v) {
			continue
		}
		s.data[keep] = v
		keep++
	}

	s.data = s.data[:keep]
}

func (s *SliceSet[U]) ToSlice() []U         { return s.data }
func (s *SliceSet[U]) ToBitSet() *BitSet[U] { return NewBitSetFrom(s.data...) }

func (s *SliceSet[U]) Ands(others ...*SliceSet[U]) {
	originalSlice := s.data
	if len(originalSlice) == 0 || len(others) == 0 {
		return
	}

	for _, o := range others {
		if len(o.data) == 0 {
			clear(originalSlice)
			s.data = s.data[:0]
			return
		}
	}

	for _, o := range others {
		sa := s.data
		so := o.data

		if len(sa) == 0 {
			break
		}

		if &sa[0] == &so[0] {
			continue
		}

		i, j, writeIdx := 0, 0, 0

		for i < len(sa) && j < len(so) {
			av, ov := sa[i], so[j]

			if av < ov {
				i++
			} else if av > ov {
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

	if len(s.data) < len(originalSlice) {
		clear(originalSlice[len(s.data):])
	}
}

func (s *SliceSet[U]) Ors(others ...*SliceSet[U]) {
	if len(others) == 0 {
		return
	}

	totalLen := len(s.data)
	for _, o := range others {
		totalLen += len(o.data)
	}

	if totalLen == len(s.data) {
		return
	}

	if cap(s.data) >= totalLen {
	} else {
		newData := make([]U, len(s.data), totalLen)
		copy(newData, s.data)
		s.data = newData
	}

	for _, o := range others {
		s.data = append(s.data, o.data...)
	}

	slices.Sort(s.data)

	res := slices.Compact(s.data)

	clear(s.data[len(res):])

	s.data = res
}
