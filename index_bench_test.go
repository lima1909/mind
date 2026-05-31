package mind

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
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

func BenchmarkRanges(b *testing.B) {

	ds := 3_000_000
	l := NewList[uint8]()
	require.NoError(b, l.CreateIndex("range", NewRangeIndex(FromValue[uint8]())))
	require.NoError(b, l.CreateIndex("rangeenc", NewRangeEncodedIndex(FromValue[uint8](), 101)))
	require.NoError(b, l.CreateIndex("fenwick", NewFenwickIndex(FromValue[uint8](), 101)))
	require.NoError(b, l.CreateIndex("sorted", NewSortedIndex(FromValue[uint8]())))

	for i := range ds {
		l.Insert(uint8(i % 100))
	}
	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func() int
		count int
	}{
		// --- RangeIndex ---
		{
			"range_<40",
			func() int {
				count, err := l.Query(Lt("range", 40)).Count()
				require.NoError(b, err)
				return count
			},
			1_200_000,
		},
		{
			"range_>60",
			func() int {
				count, err := l.Query(Gt("range", 60)).Count()
				require.NoError(b, err)
				return count
			},
			1_170_000,
		},
		{
			"range_>40<60",
			func() int {
				count, err := l.Query(Between("range", 40, 60)).Count()
				require.NoError(b, err)
				return count
			},
			630_000,
		},

		// --- SortedIndex ---
		{
			"sorted_<40",
			func() int {
				count, err := l.Query(Lt("sorted", 40)).Count()
				require.NoError(b, err)
				return count
			},
			1_200_000,
		},
		{
			"sorted_>60",
			func() int {
				count, err := l.Query(Gt("sorted", 60)).Count()
				require.NoError(b, err)
				return count
			},
			1_170_000,
		},
		{
			"sorted_>40<60",
			func() int {
				count, err := l.Query(Between("sorted", 40, 60)).Count()
				require.NoError(b, err)
				return count
			},
			630_000,
		},

		// --- FenwickIndex ---
		{
			"fenwick_<40",
			func() int {
				count, err := l.Query(Lt("fenwick", 40)).Count()
				require.NoError(b, err)
				return count
			},
			1_200_000,
		},
		{
			"fenwick_>60",
			func() int {
				count, err := l.Query(Gt("fenwick", 60)).Count()
				require.NoError(b, err)
				return count
			},
			1_170_000,
		},
		{
			"fenwick_>40<60",
			func() int {
				count, err := l.Query(Between("fenwick", 40, 60)).Count()
				require.NoError(b, err)
				return count
			},
			630_000,
		},

		// --- RangeEncodedIndex ---
		{
			"rangeenc_<40",
			func() int {
				count, err := l.Query(Lt("rangeenc", 40)).Count()
				require.NoError(b, err)
				return count
			},
			1_200_000,
		},
		{
			"rangeenc_>60",
			func() int {
				count, err := l.Query(Gt("rangeenc", 60)).Count()
				require.NoError(b, err)
				return count
			},
			1_170_000,
		},
		{
			"rangeenc_>40<60",
			func() int {
				count, err := l.Query(Between("rangeenc", 40, 60)).Count()
				require.NoError(b, err)
				return count
			},
			630_000,
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				bcount := bench.bmark()
				if bcount != bench.count {
					b.Fatalf("expected: %d, got: %d", bench.count, bcount)
				}
			}
		})
	}
}
