package mind

import "iter"

const (
	freeListEnd = -1
	occupied    = -2
)

// Slot holds the data or the index to the next free space
type slot[T any] struct {
	value    T
	nextFree int32 // If free, points to next free slot (or -1). If occupied, set to -2.
}

// FreeList don't delete an Item, instead mark it as not occupied.
// With one of the Compact Methods, you can remove thes palceholders and make the list smaller.
type FreeList[T any] struct {
	slots    []slot[T]
	freeHead int32 // Index of the first free slot (-1 if none)
	len      int
}

func NewFreeList[T any]() FreeList[T] {
	return FreeList[T]{
		slots:    make([]slot[T], 0),
		freeHead: freeListEnd, // -1 means "No free slots, append new ones"
	}
}

func NewFreeListWithCapacity[T any](capacity int) FreeList[T] {
	return FreeList[T]{
		slots:    make([]slot[T], 0, capacity),
		freeHead: freeListEnd, // -1 means "No free slots, append new ones"
	}
}

// Insert an Item to the end of the List or use a free slot, to add this item
func (l *FreeList[T]) Insert(item T) int {
	l.len++

	// no free slots in the list, append to the end
	if l.freeHead == freeListEnd {
		idx := len(l.slots)
		l.slots = append(l.slots, slot[T]{
			value:    item,
			nextFree: occupied,
		})
		return idx
	}

	idx := l.freeHead
	l.freeHead = l.slots[idx].nextFree
	l.slots[idx] = slot[T]{
		value:    item,
		nextFree: occupied,
	}

	return int(idx)
}

// Update replaced the Item on the given index, with the given Item. Return the old Item and true.
// If the given index has no Item, then will the return the zero Value of T and false.
// index must be >=0 and < len(slots), otherwise return Update zero value and false and do nothing.
func (l *FreeList[T]) Update(index int, newItem T) (T, bool) {
	if index < 0 || index >= len(l.slots) || l.slots[index].nextFree != occupied {
		var null T
		return null, false
	}

	oldItem := l.slots[index].value
	l.slots[index].value = newItem
	return oldItem, true
}

// Remove mark the Item on the given index as deleted.
// index must be >=0 and < len(slots), otherwise return Remove the zero-object and false and do nothing.
// If it is true than return the removed Item and true
func (l *FreeList[T]) Remove(index int) (T, bool) {
	var zero T
	if index < 0 || index >= len(l.slots) || l.slots[index].nextFree != occupied {
		return zero, false
	}

	result := l.slots[index].value

	// clear the value to prevent memory leaks
	l.slots[index].value = zero

	// make this slot point to the current head
	l.slots[index].nextFree = l.freeHead
	// make this slot the new head
	l.freeHead = int32(index)

	l.len--
	return result, true
}

// Get the Item on the given index, or the zero value and false, if it not exist.
// index must be >=0 and < len(slots), otherwise return Get zero value and false and do nothing.
func (l *FreeList[T]) Get(index int) (T, bool) {
	if uint(index) >= uint(len(l.slots)) || l.slots[index].nextFree != occupied {
		var null T
		return null, false
	}

	return l.slots[index].value, true
}

// Len returns the count of the occupied slots
func (l *FreeList[T]) Len() int { return l.len }

// Iter create an Iterator, to iterate over all saved Indices and Items
func (l *FreeList[T]) Iter() iter.Seq2[int, *T] {
	return func(yield func(int, *T) bool) {
		for i := range l.slots {
			if l.slots[i].nextFree == occupied {
				if !yield(i, &l.slots[i].value) {
					return
				}
			}
		}
	}
}

func (l *FreeList[T]) Values(yield func(int, T) bool) {
	for i := range l.slots {
		if slot := &l.slots[i]; slot.nextFree == occupied {
			if !yield(i, slot.value) {
				return
			}
		}
	}
}

func (l *FreeList[T]) ToValues() []T {
	result := make([]T, 0, l.Len())

	for _, slot := range l.slots {
		if slot.nextFree == occupied {
			result = append(result, slot.value)
		}
	}

	return result
}

// Clear removes all elements but keeps the allocated memory capacity.
func (l *FreeList[T]) Clear() {
	var zero slot[T]
	for i := range l.slots {
		l.slots[i] = zero
	}
	l.slots = l.slots[:0]
	l.freeHead = freeListEnd
	l.len = 0
}

// WARNING: GetMany ignores bound checks, all idxs MUST exist
func (l *FreeList[T]) getMany(idxs []uint32, result *[]T) {
	for _, idx := range idxs {
		*result = append(*result, l.slots[idx].value)
	}
}

// WARNING: Filter ignores bound checks, all idxs MUST exist
func (l *FreeList[T]) filter(idxs []uint32, result *[]T, predicate func(*T) bool) {
	for _, idx := range idxs {
		item := &l.slots[idx].value
		if predicate(item) {
			*result = append(*result, *item)
		}
	}
}

// CompactUnstable removes not used slots. Unstable means, the Indices breaks.
func (l *FreeList[T]) CompactUnstable() {
	keep := 0
	slots := l.slots

	for i := range slots {
		if slots[i].nextFree == occupied {
			slots[keep] = slots[i]
			keep++
		}
	}

	// clear the old tail to prevent memory leaks
	var zero slot[T]
	for i := keep; i < len(slots); i++ {
		slots[i] = zero
	}

	l.slots = slots[:keep]
	l.freeHead = freeListEnd
}

// CompactLinear removes not used slote.
// If an Index has changed, yout get this Info with the Callback: onMove
func (l *FreeList[T]) CompactLinear(onMove func(oldIndex, newIndex int)) {
	var null T
	keep := 0
	slots := l.slots

	for i, s := range slots {
		if s.nextFree == occupied {
			// If the read and write pointers are different, we need to move the data
			if i != keep {
				slots[keep] = s
				onMove(i, keep)

				// clear the old slot to prevent memory leaks
				slots[i] = slot[T]{value: null, nextFree: freeListEnd}
			}
			keep++
		}
	}

	l.slots = slots[:keep]
	l.freeHead = freeListEnd
}
