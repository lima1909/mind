package index

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/lima1909/mind/lidx"
)

func BenchmarkSet_vs_BulkSet(b *testing.B) {
	type foo struct {
		val uint8
	}

	count := 3_000_000
	list := make([]*foo, count)

	for i := range count {
		list[i] = &foo{uint8(i % 255)}
	}
	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func()
	}{
		{"MAP S", func() {
			idx := NewMapIndex(func(f *foo) uint8 { return f.val })
			for i, f := range list {
				idx.Set(f, uint32(i))
			}
		}},
		{"MAP B", func() {
			idx := NewMapIndex(func(f *foo) uint8 { return f.val })
			idx.BulkSet(slices.All(list))
		}},

		// -----
		{"SOR S", func() {
			idx := NewSortedIndex(func(f *foo) uint8 { return f.val })
			for i, f := range list {
				idx.Set(f, uint32(i))
			}
		}},
		{"SOR B", func() {
			idx := NewSortedIndex(func(f *foo) uint8 { return f.val })
			idx.BulkSet(slices.All(list))
		}},

		// -----
		{"RAN S", func() {
			idx := NewRangeIndex(func(f *foo) uint8 { return f.val })
			for i, f := range list {
				idx.Set(f, uint32(i))
			}
		}},
		{"RAN B", func() {
			idx := NewRangeIndex(func(f *foo) uint8 { return f.val })
			idx.BulkSet(slices.All(list))
		}},

		// -----
		{"FEN S", func() {
			idx := NewFenwickIndex(func(f *foo) uint8 { return f.val }, 255)
			for i, f := range list {
				idx.Set(f, uint32(i))
			}
		}},
		{"FEN B", func() {
			idx := NewFenwickIndex(func(f *foo) uint8 { return f.val }, 255)
			idx.BulkSet(slices.All(list))
		}},

		// -----
		{"ENC S", func() {
			idx := NewRangeEncodedIndex(func(f *foo) uint8 { return f.val }, 255)
			for i, f := range list {
				idx.Set(f, uint32(i))
			}
		}},
		{"ENC B", func() {
			idx := NewRangeEncodedIndex(func(f *foo) uint8 { return f.val }, 255)
			idx.BulkSet(slices.All(list))
		}},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				bench.bmark()
			}
		})
	}
}

func BenchmarkIDIndex(b *testing.B) {

	ds := 3_000_000

	idmap := NewIDMapIndex(FromValue[uint32]())
	idslice := NewIDSliceIndex(FromValue[uint32]())
	resultIdx := lidx.NewRawIDsWithCapacity[uint32](ds)

	for i := range ds {
		u := uint32(i)
		idmap.Set(&u, u)
		idslice.Set(&u, u)
		resultIdx.Set(u)
	}

	// random ACL IDs
	numAcls := 3_000
	acls := make([]uint32, 0, numAcls)
	aclIDs := lidx.NewRawIDsWithCapacity[uint32](numAcls)
	for range numAcls {
		id := rand.Intn(ds)
		acls = append(acls, uint32(id))
		aclIDs.Set(uint32(id))
	}

	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func()
	}{
		{
			"map",
			func() {
				for _, id := range acls {
					idx, _ := idmap.GetIndex(id)
					resultIdx.Contains(idx)
				}
			},
		},
		{
			"slice",
			func() {
				for _, id := range acls {
					idx, _ := idslice.GetIndex(id)
					resultIdx.Contains(idx)
				}
			},
		},
		{
			"prepareC",
			func() {
				r := resultIdx.Copy()
				r.And(aclIDs)
			},
		},
		{
			"prepare",
			func() {
				resultIdx.And(aclIDs)
			},
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				bench.bmark()
			}
		})
	}
}
