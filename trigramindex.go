package mind

import (
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

func (ti *TrigramIndex) Get(s string) (*RawIDs32, bool) {
	slen := len(s)

	switch len(s) {
	case 0:
		return NewRawIDs[uint32](), true
	case 1:
		bs, ok := ti.rawIDs[pack(0, 0, s[0])]
		if !ok {
			return NewRawIDs[uint32](), true
		}
		return bs, false
	case 2:
		bs, ok := ti.rawIDs[pack(0, s[0], s[1])]
		if !ok {
			return NewRawIDs[uint32](), true
		}
		return bs, false
	case 3:
		bs, ok := ti.rawIDs[pack(s[0], s[1], s[2])]
		if !ok {
			return NewRawIDs[uint32](), true
		}
		return bs, false

	default:
		var allIDs []*RawIDs32
		// collect all raw IDs
		for i := 0; i < slen-2; i++ {
			tri := pack(s[i], s[i+1], s[i+2])
			entry, ok := ti.rawIDs[tri]
			if !ok {
				// break, is NOT a substring
				return NewRawIDs[uint32](), true
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

		// false-positive filter —
		result.Removes(func(id uint32) bool {
			str := ti.buckets[id].str
			return len(str) < slen || !strings.Contains(str, s)
		})

		return result, true
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
	for j := range slen {
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
	for j := range slen {
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

// Like returns all indexed strings that match the SQL LIKE pattern.
// '%' matches any sequence of characters:
// - '%' or '%%' => all
// - 'abc' => equals
// - 'ab%' => startsWith (prefix): 'ab'
// - '%ab' => endsWith (suffix): 'ab'
// - '%ab%' => contains: 'ab'
// - '%ab%cd%' => contains: 'ab' and 'cd', in this order
// - ” => empty, reutrn the empty IDs
func (ti *TrigramIndex) Like(pattern string) (*RawIDs32, bool) {
	// empty string
	if len(pattern) == 0 {
		return NewRawIDs[uint32](), true
	}

	var parts []string
	start := 0
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '%' {
			// add only not empty parts
			if i > start {
				parts = append(parts, pattern[start:i])
			}
			start = i + 1
		}
	}

	// add the last one
	if start < len(pattern) {
		parts = append(parts, pattern[start:])
	}

	// LIKE starts here
	switch len(parts) {
	case 0:
		// ONLY‑wildcard: %, %% → full table scan
		result := NewRawIDs[uint32]()
		for i, b := range ti.buckets {
			if b.occupied {
				result.Set(uint32(i))
			}
		}
		return result, true
	case 1:
		anchoredStart := pattern[0] != '%'
		anchoredEnd := pattern[len(pattern)-1] != '%'

		part := parts[0]
		bs, canMutate := ti.Get(part)
		if bs.IsEmpty() {
			return NewRawIDs[uint32](), true
		}

		switch {
		// Equals
		case anchoredStart && anchoredEnd:
			if !canMutate {
				bs = bs.Copy()
			}
			bs.Removes(func(id uint32) bool {
				s := ti.buckets[id].str
				if len(s) < len(part) {
					return true
				}
				return s != part
			})
			return bs, true
		// StartsWith
		case anchoredStart:
			if !canMutate {
				bs = bs.Copy()
			}
			bs.Removes(func(id uint32) bool {
				s := ti.buckets[id].str
				if len(s) < len(part) {
					return true
				}
				return !strings.HasPrefix(s, part)
			})
			return bs, true
		// EndsWith
		case anchoredEnd:
			if !canMutate {
				bs = bs.Copy()
			}
			bs.Removes(func(id uint32) bool {
				s := ti.buckets[id].str
				if len(s) < len(part) {
					return true
				}
				return !strings.HasSuffix(s, part)
			})
			return bs, true
		// Contains
		default:
			return bs, canMutate
		}

	case 2:
		part1, part2 := parts[0], parts[1]

		get1, canMutate1 := ti.Get(part1)
		get2, canMutate2 := ti.Get(part2)
		count1 := get1.Count()
		count2 := get2.Count()

		// copy and use the smaller one
		var result *RawIDs32
		if count1 < count2 {
			if canMutate1 {
				result = get1
			} else {
				result = get1.Copy()
			}
			result.And(get2)
		} else {
			if canMutate2 {
				result = get2
			} else {
				result = get2.Copy()
			}
			result.And(get1)
		}

		anchoredStart := pattern[0] != '%'
		anchoredEnd := pattern[len(pattern)-1] != '%'
		totalLen := len(part1) + len(part2)

		// remove the false-positive
		result.Removes(func(id uint32) bool {
			s := ti.buckets[id].str
			if len(s) < totalLen {
				return true
			}

			// Part 1
			pos := 0
			if anchoredStart {
				if !strings.HasPrefix(s, part1) {
					return true
				}
				pos = len(part1)
			} else {
				idx := strings.Index(s, part1)
				if idx == -1 {
					return true
				}
				pos = idx + len(part1)
			}

			// Part 2
			if anchoredEnd {
				return !strings.HasSuffix(s, part2) || len(s)-len(part2) < pos
			}

			// Not end‑anchored → part2 must appear somewhere after part1
			return !strings.Contains(s[pos:], part2)
		})

		return result, true

	default:
		var reqBuf [8]*RawIDs32
		required := reqBuf[:0]

		// calculate total string length of all parts
		totalLen := 0
		// fast-Scan and extract only the most selective bitsets
		for _, part := range parts {
			totalLen += len(part)
			switch len(part) {
			case 1:
				key := pack(0, 0, part[0])
				if bs, ok := ti.rawIDs[key]; ok {
					required = append(required, bs)
				} else {
					return NewRawIDs[uint32](), true
				}
			case 2:
				key := pack(0, part[0], part[1])
				if bs, ok := ti.rawIDs[key]; ok {
					required = append(required, bs)
				} else {
					return NewRawIDs[uint32](), true
				}
			default: // >= 3
				// For long chunks, don't grab every single trigram step blindly.
				// Just grab the first and last trigram of the chunk to maximize coverage
				// while minimizing intersection CPU work.
				keyFirst := pack(part[0], part[1], part[2])
				bsFirst, ok1 := ti.rawIDs[keyFirst]
				if !ok1 {
					return NewRawIDs[uint32](), true
				}
				required = append(required, bsFirst)

				if len(part) > 3 {
					n := len(part)
					keyLast := pack(part[n-3], part[n-2], part[n-1])
					if bsLast, ok2 := ti.rawIDs[keyLast]; ok2 {
						required = append(required, bsLast)
					} else {
						return NewRawIDs[uint32](), true
					}
				}
			}
		}

		// find the smallest bitset first to minimize cloning costs
		minIdx := 0
		minCount := required[0].Count()
		for i := 1; i < len(required); i++ {
			c := required[i].Count()
			if c < minCount {
				minCount = c
				minIdx = i
			}
		}

		result := required[minIdx].Copy()

		// Intersect remaining elements
		for i, bs := range required {
			if i == minIdx {
				continue
			}
			result.And(bs)
			if result.IsEmpty() {
				return result, true
			}
		}

		anchoredStart := pattern[0] != '%'
		anchoredEnd := pattern[len(pattern)-1] != '%'

		// remove false-positive
		lastIdx := len(parts) - 1
		result.Removes(func(id uint32) bool {
			s := ti.buckets[id].str

			// if the string s is shorter as all pattern length, than can delete
			if len(s) < totalLen {
				return true
			}

			pos := 0
			for i, part := range parts {
				if i == 0 && anchoredStart {
					if !strings.HasPrefix(s, part) {
						return true
					}
					pos = len(part)
					continue
				}
				if i == lastIdx && anchoredEnd {
					return !strings.HasSuffix(s, part) || len(s)-len(part) < pos
				}

				idx := strings.Index(s[pos:], part)
				if idx == -1 {
					return true
				}
				pos += idx + len(part)
			}

			return false
		})

		return result, true
	}
}
