package main

import (
	"strings"
)

type sbucket struct {
	s        string
	occupied bool
}

type TrigramIndex struct {
	index   map[uint32]BitSet[uint32]
	buckets []sbucket
	len     int
}

func NewTrigramIndex(s ...string) TrigramIndex {
	ti := TrigramIndex{
		index:   make(map[uint32]BitSet[uint32], len(s)),
		buckets: make([]sbucket, 0, len(s)),
	}

	for i, el := range s {
		ti.Put(el, i)
	}

	return ti
}

func (ti *TrigramIndex) Get(query string) *BitSet[uint32] {
	resultBS := NewBitSet[uint32]()

	if len(query) < 3 {
		// full table scan
		for i, b := range ti.buckets {
			if b.occupied && strings.Contains(b.s, query) {
				resultBS.Set(uint32(i))
			}
		}
		return resultBS
	}

	// generate trigrams for the query
	first := true
	for i := 0; i < len(query)-2; i++ {
		tri := pack(query[i], query[i+1], query[i+2])
		bs, ok := ti.index[tri]
		if !ok {
			// If any trigram doesn't exist, the whole substring can't exist
			return NewEmptyBitSet[uint32]()
		}
		if first {
			resultBS.Or(&bs) // seed with first trigram's candidates
			first = false
		} else {
			resultBS.And(&bs) // intersect to narrow down
		}
	}

	// verification (False Positive Check)
	// Trigrams only prove the characters exist; we must verify the order/presence
	resultBS.Values(func(i uint32) bool {
		b := ti.buckets[i]
		if b.occupied && !strings.Contains(b.s, query) {
			resultBS.UnSet(i)
		}
		return true
	})

	return resultBS
}

func (ti *TrigramIndex) Put(s string, li int) {
	if li == ti.len {
		ti.buckets = append(ti.buckets, sbucket{s: s, occupied: true})
	} else if li < ti.len {
		ti.buckets[li] = sbucket{s: s, occupied: true}
	} else {
		newBuckets := make([]sbucket, li+1)
		copy(newBuckets, ti.buckets)
		ti.buckets = newBuckets
		ti.buckets[li] = sbucket{s: s, occupied: true}
	}
	ti.len++

	for j := 0; j < len(s)-2; j++ {
		tri := pack(s[j], s[j+1], s[j+2])
		bs, found := ti.index[tri]
		if !found {
			bs = *NewEmptyBitSet[uint32]()
		}
		bs.Set(uint32(li))
		ti.index[tri] = bs
	}
}

func (ti *TrigramIndex) Delete(li int) bool {
	if li >= len(ti.buckets) {
		return false
	}

	s := ti.buckets[li].s

	for j := 0; j < len(s)-2; j++ {
		tri := pack(s[j], s[j+1], s[j+2])
		if bs, found := ti.index[tri]; found {
			bs.UnSet(uint32(li))
			ti.index[tri] = bs
			bs.Shrink()
		}
	}
	ti.buckets[li] = sbucket{s: ""}
	ti.len--

	return true
}

func (ti *TrigramIndex) Len() int { return ti.len }

// pack converts 3 bytes into a single uint32 to save memory and speed up lookups
//
//go:inline
func pack(a, b, c byte) uint32 { return uint32(a)<<16 | uint32(b)<<8 | uint32(c) }
