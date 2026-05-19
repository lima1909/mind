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
	//   unigrams:   pack(0, 0, a)
	//   bigrams:    pack(0, a, b)
	//   trigrams:   pack(a, b, c)
	rawIDs  map[uint32]*RawIDs32
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
		rawIDs:  make(map[uint32]*RawIDs32, size),
		buckets: make([]strBucket, 0, size),
	}
}

func (ti *TrigramIndex) Get(s string) *RawIDs32 {
	slen := len(s)

	switch len(s) {
	case 0:
		return NewRawIDs[uint32]()
	case 1:
		bs, ok := ti.rawIDs[pack(0, 0, s[0])]
		if !ok {
			return NewRawIDs[uint32]()
		}
		return bs
	case 2:
		bs, ok := ti.rawIDs[pack(0, s[0], s[1])]
		if !ok {
			return NewRawIDs[uint32]()
		}
		return bs
	case 3:
		bs, ok := ti.rawIDs[pack(s[0], s[1], s[2])]
		if !ok {
			return NewRawIDs[uint32]()
		}
		return bs

	default:
		var allIDs []*RawIDs32
		// collect all raw IDs
		for i := 0; i < slen-2; i++ {
			tri := pack(s[i], s[i+1], s[i+2])
			entry, ok := ti.rawIDs[tri]
			if !ok {
				// break, is NOT a substring
				return NewRawIDs[uint32]()
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
	if slen == 0 {
		return // nothing to pack
	}

	// Bounds Check Elimination hint
	_ = s[slen-1]

	// unigrams (a) -> pack(0,0,a)
	for j := 0; j < slen; j++ {
		uni := pack(0, 0, s[j])
		bs, found := ti.rawIDs[uni]
		if !found {
			bs = NewRawIDs[uint32]()
			ti.rawIDs[uni] = bs
		}
		bs.Set(uint32(li))
	}

	// bigrams (ab) -> pack(0,a,b)
	for j := 0; j < slen-1; j++ {
		bi := pack(0, s[j], s[j+1])
		bs, found := ti.rawIDs[bi]
		if !found {
			bs = NewRawIDs[uint32]()
			ti.rawIDs[bi] = bs
		}
		bs.Set(uint32(li))
	}

	// trigrams
	for j := 0; j < slen-2; j++ {
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
	slen := len(s)

	// unigrams
	for j := 0; j < slen; j++ {
		uni := pack(0, 0, s[j])
		if bs, found := ti.rawIDs[uni]; found {
			bs.UnSet(uint32(li))
			if bs.Count() == 0 {
				delete(ti.rawIDs, uni)
			} else {
				bs.Shrink()
			}
		}
	}

	// bigrams
	for j := 0; j < slen-1; j++ {
		bi := pack(0, s[j], s[j+1])
		if bs, found := ti.rawIDs[bi]; found {
			bs.UnSet(uint32(li))
			if bs.Count() == 0 {
				delete(ti.rawIDs, bi)
			} else {
				bs.Shrink()
			}
		}
	}

	// trigrams
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

			// unigrams
			for j := 0; j < sLen; j++ {
				uni := pack(0, 0, s[j])
				bs := ti.rawIDs[uni]
				if bs == nil {
					bs = NewRawIDs[uint32]()
					ti.rawIDs[uni] = bs
				}
				bs.Set(uID)
			}

			// bigrams
			for j := 0; j < sLen-1; j++ {
				bi := pack(0, s[j], s[j+1])
				bs := ti.rawIDs[bi]
				if bs == nil {
					bs = NewRawIDs[uint32]()
					ti.rawIDs[bi] = bs
				}
				bs.Set(uID)
			}

			// trigrams
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
