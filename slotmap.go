package mind

import (
	"iter"
	"math"
)

// use a sentinel value to represent "No next free slot"
const sentinel = math.MaxUint32

type Handle struct {
	// the "real" index in the slotmap
	// can reused, after an index is removed
	index uint32
	// is stable and never reused
	generation uint32
}

type slotm[T any] struct {
	value T
	// generation increments every time a slot is reused.
	// this makes old handles invalid automatically.
	generation uint32
	// nextFree points to the next available slot in the chain.
	// only valid if this slot is part of the free list.
	nextFree uint32
	// to know if this is data or a free link
	occupied bool
}

type SlotMap[T any] struct {
	slots    []slotm[T]
	freeHead uint32 // Index of the first free slot
	len      int
}

func NewSlotMap[T any]() *SlotMap[T] {
	return &SlotMap[T]{
		slots:    make([]slotm[T], 0, 16), // Pre-allocate a bit
		freeHead: sentinel,
	}
}

// Add an Item to the end of the List or use a free slot, to add this item.
// If the Item is insert on a free slot, the index is reused, but the generation will be increments by one.
func (s *SlotMap[T]) Add(value T) Handle {
	var idx uint32

	// no free slots, append to the end
	if s.freeHead == sentinel {
		idx = uint32(len(s.slots))
		s.slots = append(s.slots, slotm[T]{
			value:    value,
			nextFree: sentinel,
			occupied: true,
		})
	} else {
		// pop the head from the free stack
		idx = s.freeHead
		// point head to the next one in the chain
		s.freeHead = s.slots[idx].nextFree
		// update the slot
		s.slots[idx].value = value
		s.slots[idx].occupied = true
		// generation was incremented during Remove, so it's fresh
	}

	s.len++
	return Handle{index: idx, generation: s.slots[idx].generation}
}

// Remove mark the Item on the given index as deleted.
func (s *SlotMap[T]) Remove(h Handle) bool {
	if h.index >= uint32(len(s.slots)) {
		return false
	}

	sl := &s.slots[h.index]

	// check ensures we don't double-free
	// or free a slot that has already been reused.
	if sl.generation != h.generation {
		return false
	}

	sl.generation++
	// push this slot onto the head of the free list
	sl.nextFree = s.freeHead
	s.freeHead = h.index

	// clear the value to prevent memory leaks
	var zero T
	sl.value = zero
	sl.occupied = false

	s.len--
	return true
}

// Get the Item on the given index, or the zero value and false, if it not exist.
func (s *SlotMap[T]) Get(h Handle) (T, bool) {
	if h.index >= uint32(len(s.slots)) {
		var zero T
		return zero, false
	}

	sl := &s.slots[h.index]

	// generation check
	// If the generations don't match, it means this slot was:
	// a) Removed and is now free
	// b) Removed and Reused by a different object
	if sl.generation != h.generation {
		var zero T
		return zero, false
	}

	return sl.value, true
}

func (s *SlotMap[T]) Len() int { return s.len }

// Iter create an Iterator, to iterate over all saved Indices and Items
func (s *SlotMap[T]) Iter() iter.Seq2[Handle, T] {
	return func(yield func(Handle, T) bool) {
		for i, item := range s.slots {
			if item.occupied {
				if !yield(Handle{uint32(i), item.generation}, item.value) {
					return
				}
			}
		}
	}
}

// Compact removes all free slots from the SlotMap.
// This invalidates existing handles!
// It returns a map of {OldIndex -> NewIndex} so you can fix your handles.
func (s *SlotMap[T]) Compact(move func(oldIndex, newIndex uint32)) {
	keep := 0
	slots := s.slots

	for i, s := range slots {
		if s.occupied {
			if i != keep {
				s.generation = 0
				slots[keep] = s
				move(uint32(i), uint32(keep))
			}
			keep++
		}
	}

	s.slots = s.slots[:keep]
	s.freeHead = sentinel
}
