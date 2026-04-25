package mind

import (
	"iter"
	"strings"
)

type sbucket struct {
	s        string
	occupied bool
}

type TrigramIndex struct {
	index   map[uint32]*RawIDs32
	buckets []sbucket
	len     int
}

func NewTrigramIndex() TrigramIndex { return NewTrigramIndexWithCapacity(0) }

func NewTrigramIndexWithCapacity(size int) TrigramIndex {
	return TrigramIndex{
		index:   make(map[uint32]*RawIDs32, size),
		buckets: make([]sbucket, 0, size),
	}
}

func NewTrigramIndexFrom(s ...string) TrigramIndex {
	ti := NewTrigramIndexWithCapacity(len(s))

	for i, el := range s {
		ti.Put(el, i)
	}

	return ti
}

func (ti *TrigramIndex) Get(query string) *RawIDs32 {
	result := NewRawIDs[uint32]()

	if len(query) < 3 {
		// full table scan
		for i, b := range ti.buckets {
			if b.occupied && strings.Contains(b.s, query) {
				result.Set(uint32(i))
			}
		}
		return result
	}

	// generate trigrams for the query
	first := true
	for i := 0; i < len(query)-2; i++ {
		tri := pack(query[i], query[i+1], query[i+2])
		bs, ok := ti.index[tri]
		if !ok {
			// If any trigram doesn't exist, the whole substring can't exist
			return NewRawIDs[uint32]()
		}
		if first {
			result.Or(bs) // seed with first trigram's candidates
			first = false
		} else {
			result.And(bs) // intersect to narrow down
		}
	}

	// verification (False Positive Check)
	// Trigrams only prove the characters exist; we must verify the order/presence
	result.Values(func(i uint32) bool {
		b := ti.buckets[i]
		if b.occupied && !strings.Contains(b.s, query) {
			result.UnSet(i)
		}
		return true
	})

	return result
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
			bs = NewRawIDs[uint32]()
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

func TrigramIndexBulkPut[OBJ any](ti *TrigramIndex, mapFn func(*OBJ) string, objs iter.Seq2[int, *OBJ]) {
	if len(ti.index) == 0 {
		ti.index = make(map[uint32]*RawIDs32, 1024)
	}

	for id, o := range objs {
		// expand buckets on the fly
		if id >= len(ti.buckets) {
			newSize := id + 1
			if newSize < len(ti.buckets)*2 {
				newSize = len(ti.buckets) * 2 // Exponential growth
			}
			nb := make([]sbucket, newSize)
			copy(nb, ti.buckets)
			ti.buckets = nb
		}

		s := mapFn(o)
		ti.buckets[id] = sbucket{s: s, occupied: true}
		if id >= ti.len {
			ti.len = id + 1
		}

		// build Index (No "seen" check, no temp slices)
		// Just straight map access. The CPU is very good at this.
		uID := uint32(id)
		sLen := len(s)
		for j := 0; j < sLen-2; j++ {
			tri := pack(s[j], s[j+1], s[j+2])

			bs := ti.index[tri]
			if bs == nil {
				bs = NewRawIDs[uint32]()
				ti.index[tri] = bs
			}
			bs.Set(uID)
		}
	}
}
