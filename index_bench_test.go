package mind

import (
	"slices"
	"testing"
)

func BenchmarkSet_vs_BulkSet(b *testing.B) {
	type foo struct {
		val uint8
	}

	count := 3_000_000
	list := make([]*foo, count)

	for i := range count {
		list[i] = &foo{uint8(i % 30_000)}
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
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				bench.bmark()
			}
		})
	}
}
