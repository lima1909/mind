package mind

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func BenchmarkTrigramIndex_BulkPut_vs_Put(b *testing.B) {
	ds := 3_000_000
	n := 0
	names := strings.Split(names_txt, "\n")

	start := time.Now()
	l := make([]*string, ds)

	for i := 0; i < ds; i++ {
		if n%6779 == 0 {
			n = 0
		}
		n++

		l[i] = &names[n]
	}

	fmt.Printf("- Count: %d, Time: %s\n", len(l), time.Since(start))

	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func() int
	}{
		{
			"Put",
			func() int {
				ti := NewTrigramIndexWithCapacity(ds)
				for i, s := range l {
					ti.Put(*s, i)
				}
				return ti.len
			},
		},
		{
			"Bulk",
			func() int {
				ti := NewTrigramIndexWithCapacity(ds)
				handler := SingleValueHandler[string, string]{func(s *string) string { return *s }}
				TrigramIndexBulkPut(&ti, handler, slices.All(l))
				return ti.len
			},
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			count := 0
			for b.Loop() {
				count = max(count, bench.bmark())
				if count != ds {
					b.Fatalf("expected: %d, got: %d", ds, count)
				}
			}
		})
	}
}

func BenchmarkTrigramIndex_Get(b *testing.B) {
	ds := 3_000_000
	n := 0
	names := strings.Split(names_txt, "\n")

	ti := NewTrigramIndexWithCapacity(ds)

	for i := 0; i < ds; i++ {
		if n%6770 == 0 {
			n = 0
		}
		n++

		ti.Put(names[n], i)
	}

	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func() int
		count int
	}{
		{
			"Get_ana",
			func() int {
				ids := ti.Get("ana")
				return ids.Count()
			},
			35_007,
		},
		{
			"Get_bel",
			func() int {
				ids := ti.Get("bel")
				return ids.Count()
			},
			14_629,
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				count := bench.bmark()
				if count != bench.count {
					b.Fatalf("expected: %d, got: %d", ds, count)
				}
			}
		})
	}
}
