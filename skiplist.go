package mind

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

// FindFromSortedKeys calls visit for all finding keys.
// Important: they keys slice MUST be sorted!
func (sl *SkipList[K, V]) FindFromSortedKeys(visit VisitFn[K, V], keys ...K) {
	if len(keys) == 0 {
		return
	}

	// track the last-visited node at every level so the express lanes
	var cursor [maxLevel]*node[K, V]
	for i := range sl.level {
		cursor[i] = sl.head
	}

	for _, key := range keys {
		// descend from the highest level, resuming from each level's cursor
		for i := int(sl.level) - 1; i >= 0; i-- {
			x := cursor[i]
			for x.next[i] != nil && x.next[i].key < key {
				x = x.next[i]
			}
			cursor[i] = x

			// Propagate down: if level i reached a node ahead of cursor[i-1],
			// pull cursor[i-1] forward so lower levels don't do a linear walk.
			if i > 0 && x != sl.head {
				c := cursor[i-1]
				if c == sl.head || c.key < x.key {
					cursor[i-1] = x
				}
			}
		}

		// check the candidate node at level 0
		candidate := cursor[0].next[0]
		if candidate == nil {
			return
		}

		if candidate.key == key {
			if !visit(candidate.key, candidate.value) {
				return
			}
			// advance all cursors at levels this node participates in
			for i := byte(0); i < candidate.level; i++ {
				cursor[i] = candidate
			}
		}
	}
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
