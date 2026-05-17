package mind

import (
	"iter"
	"strings"
)

// FuzzyGet exmaple:
// Stephen
// Steve
// Seven

type strBucket struct {
	str      string
	occupied bool
}

type TrigramIndex struct {
	rawIDs map[uint32]*RawIDs32 // trigrams
	// biRawIDs map[uint16]*RawIDs32 // bigrams
	buckets []strBucket
	len     int
}

func NewTrigramIndex() TrigramIndex {
	return NewTrigramIndexWithCapacity(0)
}

func NewTrigramIndexFrom(s ...string) TrigramIndex {
	ti := NewTrigramIndexWithCapacity(len(s))
	for i, el := range s {
		ti.Put(el, i)
	}
	return ti
}

func NewTrigramIndexWithCapacity(size int) TrigramIndex {
	return TrigramIndex{
		rawIDs: make(map[uint32]*RawIDs32, size),
		// biRawIDs: make(map[uint16]*RawIDs32, size),
		buckets: make([]strBucket, 0, size),
	}
}

func (ti *TrigramIndex) Get(s string) *RawIDs32 {
	result := NewRawIDs[uint32]()

	slen := len(s)

	switch slen {
	case 0:
		return result
	case 1, 2:
		// full table scan
		for i, b := range ti.buckets {
			if strings.Contains(b.str, s) {
				result.Set(uint32(i))
			}
		}
		return result
	// case 2:
	// 	if bs, ok := ti.biRawIDs[pack2(s[0], s[1])]; ok {
	// 		return bs
	// 	}
	// 	return result

	case 3:
		// Optimization for 3 letters
		bs, ok := ti.rawIDs[pack(s[0], s[1], s[2])]
		if !ok {
			return result
		}
		return bs

	default:
		var allIDs []*RawIDs32
		// collect all raw IDs
		for i := 0; i < slen-2; i++ {
			tri := pack(s[i], s[i+1], s[i+2])
			entry, ok := ti.rawIDs[tri]
			if !ok {
				return result
			}
			allIDs = append(allIDs, entry)
		}

		// smallest first
		minIdx := 0
		for i := 1; i < len(allIDs); i++ {
			if allIDs[i].Count() < allIDs[minIdx].Count() {
				minIdx = i
			}
		}

		result := allIDs[minIdx].Copy()
		for i, entry := range allIDs {
			if i == minIdx {
				continue
			}
			result.And(entry)
		}

		// false‑positive filter
		result.Values(func(id uint32) bool {
			str := ti.buckets[id].str
			if len(str) < slen || !strings.Contains(str, s) {
				result.UnSet(id)
			}
			return true
		})
		return result
	}
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

	slen := len(s)
	if slen < 3 {
		return // nothing to pack
	}

	// Bounds Check Elimination hint
	_ = s[slen-1]

	// Index bigrams (if string length >= 2)
	// if slen >= 2 {
	// 	for j := 0; j < slen-1; j++ {
	// 		bi := pack2(s[j], s[j+1])
	// 		bs, found := ti.biRawIDs[bi]
	// 		if !found {
	// 			bs = NewRawIDs[uint32]()
	// 			ti.biRawIDs[bi] = bs
	// 		}
	// 		bs.Set(uint32(li))
	// 	}
	// }

	// Index trigrams
	// if slen >= 3 {
	for j := 0; j < slen-2; j++ {
		tri := pack(s[j], s[j+1], s[j+2])
		bs, found := ti.rawIDs[tri]
		if !found {
			bs = NewRawIDs[uint32]()
			ti.rawIDs[tri] = bs
		}
		bs.Set(uint32(li))
	}
	// }
}

func (ti *TrigramIndex) Delete(li int) bool {
	if li >= len(ti.buckets) || !ti.buckets[li].occupied {
		return false
	}

	s := ti.buckets[li].str
	slen := len(s)

	// Remove from bigrams
	// if slen >= 2 {
	// 	for j := 0; j < len(s)-1; j++ {
	// 		bi := pack2(s[j], s[j+1])
	// 		if bs, found := ti.biRawIDs[bi]; found {
	// 			bs.UnSet(uint32(li))
	// 			if bs.Count() == 0 {
	// 				delete(ti.biRawIDs, bi)
	// 			} else {
	// 				bs.Shrink()
	// 			}
	// 		}
	// 	}
	// }
	//
	// // Remove from trigrams
	// if len(s) >= 3 {
	for j := 0; j < slen-2; j++ {
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
	// }

	ti.buckets[li] = strBucket{str: "", occupied: false}
	ti.len--

	return true
}

func (ti *TrigramIndex) Len() int { return ti.len }

// pack converts 3 bytes into a single uint32 to save memory and speed up lookups
//
//go:inline
func pack(a, b, c byte) uint32 { return uint32(a)<<16 | uint32(b)<<8 | uint32(c) }

// pack2 converts two bytes into a uint16 for bigram lookup.
// max 65 536 strings
//
//go:inline
// func pack2(a, b byte) uint16 { return uint16(a)<<8 | uint16(b) }

func TrigramIndexBulkPut[OBJ any](ti *TrigramIndex, vhandler SingleValueHandler[OBJ, string], objs iter.Seq2[int, *OBJ]) {
	for id, o := range objs {
		// Expand bucket slice on the fly (unchanged)
		if id >= len(ti.buckets) {
			newSize := id + 1
			if newSize < len(ti.buckets)*2 {
				newSize = len(ti.buckets) * 2
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

			uID := uint32(id)
			sLen := len(s)

			// Index trigrams (original)
			for j := 0; j < sLen-2; j++ {
				tri := pack(s[j], s[j+1], s[j+2])
				bs := ti.rawIDs[tri]
				if bs == nil {
					bs = NewRawIDs[uint32]()
					ti.rawIDs[tri] = bs
				}
				bs.Set(uID)
			}

			// Index bigrams (new)
			// for j := 0; j < sLen-1; j++ {
			// 	bi := pack2(s[j], s[j+1])
			// 	bs := ti.biRawIDs[bi]
			// 	if bs == nil {
			// 		bs = NewRawIDs[uint32]()
			// 		ti.biRawIDs[bi] = bs
			// 	}
			// 	bs.Set(uID)
			// }
		})
	}
}
