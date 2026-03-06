package mind

// It is designed for high-speed, constant-time O(1) lookups.
// It can only answer "Definitely no" or "Probably yes" (like: Bloom filters),
// but it is an alternative to the Bloom Filter because it supports deletions.
//
// https://www.vldb.org/pvldb/vol13/p3559-kipf.pdf
// https://www.youtube.com/watch?v=UOB8FJDt670

// fast hashing
// https://github.com/vvatanabe/go-sdbm

const (
	bucketSize = 4   // entries per bucket
	maxKicks   = 500 // max displacement attempts
	// loadFactor = 0.9
	defaultCap = 1 << 16
)

type entry struct {
	key   uint32
	value uint32
	used  bool
}

type bucket [bucketSize]entry

type cuckooIndex struct {
	buckets []bucket
	count   uint32
	mask    uint32
	seed1   uint32
	seed2   uint32
}

func newCuckoo() *cuckooIndex {
	return newWithCapacity(defaultCap)
}

func newWithCapacity(cap uint32) *cuckooIndex {
	// round up to power of 2
	size := uint32(1)
	for size < cap {
		size <<= 1
	}
	return &cuckooIndex{
		buckets: make([]bucket, size),
		mask:    size - 1,
		seed1:   0x9e3779b9,
		seed2:   0x85ebca6b,
	}
}

//go:nosplit
func (c *cuckooIndex) hash1(key uint32) uint32 {
	key ^= c.seed1
	key *= 0xcc9e2d51
	key ^= key >> 16
	return key & c.mask
}

//go:nosplit
func (c *cuckooIndex) hash2(key uint32) uint32 {
	key ^= c.seed2
	key *= 0x1b873593
	key ^= key >> 16
	return key & c.mask
}

func (c *cuckooIndex) Get(key uint32) (uint32, bool) {
	h1 := c.hash1(key)
	for i := range bucketSize {
		if c.buckets[h1][i].used && c.buckets[h1][i].key == key {
			return c.buckets[h1][i].value, true
		}
	}

	h2 := c.hash2(key)
	for i := range bucketSize {
		if c.buckets[h2][i].used && c.buckets[h2][i].key == key {
			return c.buckets[h2][i].value, true
		}
	}

	return 0, false
}

func (c *cuckooIndex) Put(key, value uint32) bool {
	// check if key exists
	h1 := c.hash1(key)
	for i := range bucketSize {
		if c.buckets[h1][i].used && c.buckets[h1][i].key == key {
			c.buckets[h1][i].value = value
			return true
		}
	}

	h2 := c.hash2(key)
	for i := range bucketSize {
		if c.buckets[h2][i].used && c.buckets[h2][i].key == key {
			c.buckets[h2][i].value = value
			return true
		}
	}

	// try to insert in bucket1
	for i := range bucketSize {
		if !c.buckets[h1][i].used {
			c.buckets[h1][i] = entry{key: key, value: value, used: true}
			c.count++
			return true
		}
	}

	// try to insert in bucket2
	for i := range bucketSize {
		if !c.buckets[h2][i].used {
			c.buckets[h2][i] = entry{key: key, value: value, used: true}
			c.count++
			return true
		}
	}

	// cuckoo displacement
	return c.kick(key, value, h1, 0)
}

func (c *cuckooIndex) kick(key, value uint32, idx uint32, depth int) bool {
	if depth >= maxKicks {
		// need to resize
		if c.resize() {
			return c.Put(key, value)
		}
		return false
	}

	// evict random entry from bucket
	slot := depth % bucketSize
	evicted := c.buckets[idx][slot]
	c.buckets[idx][slot] = entry{key: key, value: value, used: true}

	// find alternate bucket for evicted
	alt := c.hash1(evicted.key)
	if alt == idx {
		alt = c.hash2(evicted.key)
	}

	for i := range bucketSize {
		if !c.buckets[alt][i].used {
			c.buckets[alt][i] = evicted
			c.count++
			return true
		}
	}

	return c.kick(evicted.key, evicted.value, alt, depth+1)
}

func (c *cuckooIndex) Delete(key uint32) bool {
	h1 := c.hash1(key)
	for i := range bucketSize {
		if c.buckets[h1][i].used && c.buckets[h1][i].key == key {
			c.buckets[h1][i].used = false
			c.count--
			return true
		}
	}

	h2 := c.hash2(key)
	for i := range bucketSize {
		if c.buckets[h2][i].used && c.buckets[h2][i].key == key {
			c.buckets[h2][i].used = false
			c.count--
			return true
		}
	}

	return false
}

func (c *cuckooIndex) resize() bool {
	oldBuckets := c.buckets
	newSize := uint32(len(c.buckets)) << 1
	c.buckets = make([]bucket, newSize)
	c.mask = newSize - 1
	c.count = 0

	for _, b := range oldBuckets {
		for _, e := range b {
			if e.used {
				if !c.Put(e.key, e.value) {
					return false
				}
			}
		}
	}
	return true
}

func (c *cuckooIndex) Len() uint32 {
	return c.count
}
