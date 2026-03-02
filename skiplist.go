package main

import (
	"cmp"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// A SkipList is a data structure that allows for fast search, insertion, and deletion within a sorted list.
// It acts as an alternative to balanced binary search trees.
//
// Think of a SkipList as a standard Sorted Linked List but with "express lanes."
//
// https://en.wikipedia.org/wiki/Skip_list

const (
	maxLevel   = 16 // supports up to ~4.3 million elements
	population = 0.25
)

type VisitFn[K any, V any] func(key K, val V) bool

type node[K cmp.Ordered, V any] struct {
	key   K
	value V
	level byte
	next  [maxLevel]*node[K, V]
}

type SkipList[K cmp.Ordered, V any] struct {
	head  *node[K, V]
	level byte

	rnd *rand.Rand
}

// randomLevel generates a random height (level)
//
//go:inline
func (sl *SkipList[K, V]) randomLevel() byte {
	lvl := byte(1)
	for lvl < maxLevel && sl.rnd.Float64() < population {
		lvl++
	}
	return lvl
}

// NewSkipList creates a new SkipList
func NewSkipList[K cmp.Ordered, V any]() SkipList[K, V] {
	return SkipList[K, V]{
		head:  &node[K, V]{level: maxLevel},
		level: 1,
		rnd:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Get returns value and whether it exists
func (sl *SkipList[K, V]) Get(key K) (V, bool) {
	x := sl.head
	for i := int(sl.level) - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && next.key < key; next = x.next[i] {
			x = next
		}
	}

	x = x.next[0]
	if x != nil && x.key == key {
		// key found
		return x.value, true
	}

	var zeroVal V
	return zeroVal, false
}

// Put inserts or updates a key with the given value.
// Returns true if a new node was inserted, false if an existing key was updated.
func (sl *SkipList[K, V]) Put(key K, value V) bool {
	update := [maxLevel]*node[K, V]{}
	x := sl.head

	// search for the position and fill the 'update' array
	for i := int(sl.level) - 1; i >= 0; i-- {
		// move forward while next node's key < insertion key
		for next := x.next[i]; next != nil && next.key < key; next = x.next[i] {
			x = next
		}
		// save the last node visited at this level
		update[i] = x
	}

	// check if the key already exists
	x = x.next[0]
	if x != nil && x.key == key {
		x.value = value // update existing value
		return false    // not a new insertion
	}

	// key does not exist, prepare new node level
	lvl := sl.randomLevel()

	// if the new level is higher than current, initialize 'update' for the gap
	if lvl > sl.level {
		for i := sl.level; i < lvl; i++ {
			update[i] = sl.head
		}
	}

	// create and link the new node
	n := &node[K, V]{key: key, value: value, level: lvl}

	for i := range lvl {
		n.next[i] = update[i].next[i]
		update[i].next[i] = n
	}

	// update global level
	if lvl > sl.level {
		sl.level = lvl
	}

	return true
}

// Delete removes the value for a given key
// If the key was not found: false, otherwise true, if the key was deleted.
func (sl *SkipList[K, V]) Delete(key K) bool {
	update := [maxLevel]*node[K, V]{}
	x := sl.head
	for i := int(sl.level) - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && next.key < key; next = x.next[i] {
			x = next
		}
		update[i] = x
	}

	x = x.next[0]
	if x == nil || x.key != key {
		// not found, no value deleted
		return false
	}

	for i := byte(0); i < sl.level; i++ {
		if update[i].next[i] != x {
			break
		}
		update[i].next[i] = x.next[i]
	}

	for sl.level > 1 && sl.head.next[sl.level-1] == nil {
		sl.level--
	}

	return true
}

// Traverse over the complete Skiplist and calling the visitor
// the return value false means, not to the end, otherwise true
func (sl *SkipList[K, V]) Traverse(visit VisitFn[K, V]) bool {
	for next := sl.head.next[0]; next != nil; next = next.next[0] {
		// break if false (simulate yield)
		if !visit(next.key, next.value) {
			// reads not to the end
			return false
		}
	}

	return true
}

// FindSortedKeys calls visit for all finding keys.
// Important: they keys slice MUST be sorted!
func (sl *SkipList[K, V]) FindSortedKeys(visit VisitFn[K, V], keys ...K) {
	if len(keys) == 0 {
		return
	}

	curr := sl.head

	for _, key := range keys {
		for i := int(sl.level) - 1; i >= 0; i-- {
			for next := curr.next[i]; next != nil && next.key < key; next = curr.next[i] {
				curr = next
			}
		}

		x := curr.next[0]
		if x != nil && x.key == key {
			if !visit(x.key, x.value) {
				return
			}
		}
	}
}

// FindMaybeSortedKeys  calls visit for all finding keys.
// They keys slice MUST not be sorted, but the performance is better, if they are
func (sl *SkipList[K, V]) FindMaybeSortedKeys(visit VisitFn[K, V], keys ...any) error {
	if len(keys) == 0 {
		return nil
	}

	lastK, err := ValueFromAny[K](keys[0])
	if err != nil {
		return err
	}

	key := lastK
	curr := sl.head
	for i, k := range keys {
		// do not compute the first key again
		if i != 0 {
			key, err = ValueFromAny[K](k)
			if err != nil {
				return err
			}

			// reset the head, if an key is less than the previous
			if key < lastK {
				curr = sl.head
			}
		}

		for i := int(sl.level) - 1; i >= 0; i-- {
			for next := curr.next[i]; next != nil && next.key < key; next = curr.next[i] {
				curr = next
			}
		}

		x := curr.next[0]
		if x != nil && x.key == key {
			if !visit(x.key, x.value) {
				return nil
			}
		}

		// set new last key
		lastK = key
	}

	return nil
}

// Range traverse 'from' until 'to' over Skiplist and calling the visitor
func (sl *SkipList[K, V]) Range(from, to K, visit VisitFn[K, V]) {
	if from > to {
		return
	}

	// find the first node >= from
	x := sl.head
	for i := int(sl.level) - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && next.key < from; next = x.next[i] {
			x = next
		}
	}

	// move to the actual first node at level 0
	x = x.next[0]
	if x == nil || x.key > to {
		return
	}

	// collect all nodes until we exceed 'to'
	for x != nil && x.key <= to {
		if !visit(x.key, x.value) {
			return
		}
		x = x.next[0] // Always stay on the ground floor (Level 0)
	}
}

// Less calls visit for all keys < the given key
func (sl *SkipList[K, V]) Less(key K, visit VisitFn[K, V]) {
	// start directly at the first element (Level 0)
	for x := sl.head.next[0]; x != nil && x.key < key; x = x.next[0] {
		// stop immediately if the visitor returns false
		if !visit(x.key, x.value) {
			return
		}
	}
}

// LessEqual calls visit for all keys <= the given key
func (sl *SkipList[K, V]) LessEqual(key K, visit VisitFn[K, V]) {
	// start directly at the first element (Level 0)
	for x := sl.head.next[0]; x != nil && x.key <= key; x = x.next[0] {
		if !visit(x.key, x.value) {
			return
		}
	}
}

// Greater calls visit for all keys > the given key
func (sl *SkipList[K, V]) Greater(key K, visit VisitFn[K, V]) {
	x := sl.head

	// jump to the starting point
	for i := int(sl.level) - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && next.key < key; next = x.next[i] {
			x = next
		}
	}

	// walk right until the end of the list
	for x = x.next[0]; x != nil; x = x.next[0] {
		if x.key == key {
			continue
		}
		if !visit(x.key, x.value) {
			return
		}
	}
}

// GreaterEqual calls visit for all keys >= the given key
func (sl *SkipList[K, V]) GreaterEqual(key K, visit VisitFn[K, V]) {
	x := sl.head

	// jump to the starting point
	for i := int(sl.level) - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && next.key < key; next = x.next[i] {
			x = next
		}
	}

	// walk right until the end of the list
	for x = x.next[0]; x != nil; x = x.next[0] {
		if !visit(x.key, x.value) {
			return
		}
	}
}

// StringStartsWith finds all keys with the given prefix.
// If prefix (K) is not a string, this method panics!
func (sl *SkipList[K, V]) StringStartsWith(prefix K, visit VisitFn[K, V]) bool {
	prefixStr, ok := any(prefix).(string)
	if !ok {
		panic(fmt.Sprintf("StringStartsWith supports only strings, not: %T", prefix))
	}

	x := sl.head
	for i := int(sl.level) - 1; i >= 0; i-- {
		for next := x.next[i]; next != nil && next.key < prefix; next = x.next[i] {
			x = next
		}
	}

	for x = x.next[0]; x != nil; x = x.next[0] {
		// We still have to box x.key here, but we stop as soon as it mismatches
		kStr := any(x.key).(string)
		if !strings.HasPrefix(kStr, prefixStr) {
			break
		}
		if !visit(x.key, x.value) {
			break
		}
	}

	return true
}

// MinKey returns the first (smallest) Key
// or the zero value and false, if the list is empty.
func (sl *SkipList[K, V]) MinKey() (K, bool) {
	// The first node on the bottom level (level 0) is the minimum
	first := sl.head.next[0]
	if first == nil {
		var zero K
		return zero, false
	}

	return first.key, true
}

// MaxKey returns the last (biggest) Key
// or the zero value and false, if the list is empty.
func (sl *SkipList[K, V]) MaxKey() (K, bool) {
	x := sl.head
	// start at the highest lane and jump as far right as possible
	for i := int(sl.level) - 1; i >= 0; i-- {
		for x.next[i] != nil {
			x = x.next[i]
		}
	}

	// list is empty
	if x == sl.head {
		var zero K
		return zero, false
	}

	return x.key, true
}

// FirstValue returns the value associated with the smallest key
// or the zero value and false, if the list is empty.
func (sl *SkipList[K, V]) FirstValue() (V, bool) {
	// The first node on the bottom level (level 0) is the minimum
	first := sl.head.next[0]
	if first == nil {
		var zero V
		return zero, false
	}

	return first.value, true
}

// LastValue returns the value associated with the largest key
// or the zero value and false, if the list is empty.
func (sl *SkipList[K, V]) LastValue() (V, bool) {
	x := sl.head
	// start at the highest lane and jump as far right as possible
	for i := int(sl.level) - 1; i >= 0; i-- {
		for x.next[i] != nil {
			x = x.next[i]
		}
	}

	// list is empty
	if x == sl.head {
		var zero V
		return zero, false
	}

	return x.value, true
}
