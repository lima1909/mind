package mind

import (
	"iter"
	"strings"
)

type strBucket struct {
	str      string
	occupied bool
}

type TrigramIndex struct {
	rawIDs  map[uint32]*RawIDs32
	buckets []strBucket
	len     int
}

func NewTrigramIndex() TrigramIndex { return NewTrigramIndexWithCapacity(0) }

func NewTrigramIndexWithCapacity(size int) TrigramIndex {
	return TrigramIndex{
		rawIDs:  make(map[uint32]*RawIDs32, size),
		buckets: make([]strBucket, 0, size),
	}
}

func NewTrigramIndexFrom(s ...string) TrigramIndex {
	ti := NewTrigramIndexWithCapacity(len(s))

	for i, el := range s {
		ti.Put(el, i)
	}

	return ti
}

func (ti *TrigramIndex) Get(s string) *RawIDs32 {
	result := NewRawIDs[uint32]()

	if len(s) < 3 {
		// full table scan
		for i, b := range ti.buckets {
			if b.occupied && strings.Contains(b.str, s) {
				result.Set(uint32(i))
			}
		}
		return result
	}

	// generate trigrams for the query
	first := true
	for i := 0; i < len(s)-2; i++ {
		tri := pack(s[i], s[i+1], s[i+2])
		bs, ok := ti.rawIDs[tri]
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

	// fast path: Skip verification if length is exactly 3!
	if len(s) == 3 {
		return result
	}

	// verification (False Positive Check)
	// Trigrams only prove the characters exist; we must verify the order/presence
	result.Values(func(i uint32) bool {
		b := ti.buckets[i]
		if b.occupied && !strings.Contains(b.str, s) {
			result.UnSet(i)
		}
		return true
	})

	return result
}

func (ti *TrigramIndex) Put(s string, li int) {
	wasOccupied := false
	if li < len(ti.buckets) {
		wasOccupied = ti.buckets[li].occupied
	} else {
		if cap(ti.buckets) <= li {
			newBuckets := make([]strBucket, li+1, (li+1)*2) // double capacity to avoid frequent allocations
			copy(newBuckets, ti.buckets)
			ti.buckets = newBuckets
		} else {
			ti.buckets = ti.buckets[:li+1]
		}
	}

	ti.buckets[li] = strBucket{str: s, occupied: true}
	if !wasOccupied {
		ti.len++
	}

	// Bounds Check Elimination hint
	if len(s) < 3 {
		return // nothing to pack
	}
	_ = s[len(s)-1]

	for j := 0; j < len(s)-2; j++ {
		tri := pack(s[j], s[j+1], s[j+2])
		bs, found := ti.rawIDs[tri]
		if !found {
			bs = NewRawIDs[uint32]()
			ti.rawIDs[tri] = bs
		}
		bs.Set(uint32(li))
	}
}

func (ti *TrigramIndex) Delete(li int) bool {
	if li >= len(ti.buckets) || !ti.buckets[li].occupied {
		return false
	}

	s := ti.buckets[li].str

	for j := 0; j < len(s)-2; j++ {
		tri := pack(s[j], s[j+1], s[j+2])
		if bs, found := ti.rawIDs[tri]; found {
			bs.UnSet(uint32(li))

			if bs.Count() == 0 {
				delete(ti.rawIDs, tri)
			} else {
				bs.Shrink()
			}
		}
	}

	ti.buckets[li] = strBucket{str: "", occupied: false}
	ti.len--

	return true
}

func (ti *TrigramIndex) Len() int { return ti.len }

// pack converts 3 bytes into a single uint32 to save memory and speed up lookups
//
//go:inline
func pack(a, b, c byte) uint32 { return uint32(a)<<16 | uint32(b)<<8 | uint32(c) }

func TrigramIndexBulkPut[OBJ any](ti *TrigramIndex, vhandler SingleValueHandler[OBJ, string], objs iter.Seq2[int, *OBJ]) {
	if len(ti.rawIDs) == 0 {
		ti.rawIDs = make(map[uint32]*RawIDs32, 1024)
	}

	for id, o := range objs {
		// expand buckets on the fly
		if id >= len(ti.buckets) {
			newSize := id + 1
			if newSize < len(ti.buckets)*2 {
				newSize = len(ti.buckets) * 2 // Exponential growth
			}
			nb := make([]strBucket, newSize)
			copy(nb, ti.buckets)
			ti.buckets = nb
		}

		vhandler.Handle(o, func(s string) {
			ti.buckets[id] = strBucket{str: s, occupied: true}
			if id >= ti.len {
				ti.len = id + 1
			}

			// build Index (No "seen" check, no temp slices)
			// Just straight map access. The CPU is very good at this.
			uID := uint32(id)
			sLen := len(s)
			for j := 0; j < sLen-2; j++ {
				tri := pack(s[j], s[j+1], s[j+2])

				bs := ti.rawIDs[tri]
				if bs == nil {
					bs = NewRawIDs[uint32]()
					ti.rawIDs[tri] = bs
				}
				bs.Set(uID)
			}
		})
	}
}
