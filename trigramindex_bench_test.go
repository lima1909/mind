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

	for i := range ds {
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
				for id, o := range slices.All(l) {
					handler.Handle(o, func(s string) {
						ti.Put(s, id)
					})
				}
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
	names := strings.Split(names_txt, "\n")

	// Clean out empty lines to avoid indexing garbage data
	var validNames []string
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			validNames = append(validNames, trimmed)
		}
	}

	if len(validNames) == 0 {
		b.Fatal("names_txt contains no valid data to index")
	}

	// 1. Bulk setup phase
	ti := NewTrigramIndexWithCapacity(ds)
	for i := range ds {
		// Clean, allocation-free round-robin data selection
		ti.Put(validNames[i%len(validNames)], i)
	}

	bmarks := []struct {
		name  string
		query string
		count int
	}{
		{"Get____y", "y", 523_516},
		{"Get___na", "na", 200_027},
		{"Get__ana", "ana", 34_958},
		{"Get_anai", "anai", 442},
	}

	// Global variable assignment target to prevent
	// aggressive compiler Dead Code Elimination (DCE)
	var globalCount int

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			// If you had setup work *inside* the sub-benchmark loop,
			// you would call b.ResetTimer() right here.

			for b.Loop() {
				ids, _ := ti.Get(bench.query)
				count := ids.Count()

				if count != bench.count {
					b.Fatalf("%s: expected count %d, got %d", bench.name, bench.count, count)
				}

				globalCount = count // Avoid compiler optimizations safely
			}
		})
	}

	// Reference global count outside to ensure it's never optimized out
	_ = globalCount
}

func BenchmarkTrigramIndex_Like(b *testing.B) {
	ds := 3_000_000
	names := strings.Split(names_txt, "\n")

	var validNames []string
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			validNames = append(validNames, trimmed)
		}
	}

	if len(validNames) == 0 {
		b.Fatal("names_txt contains no valid data to index")
	}

	ti := NewTrigramIndexWithCapacity(ds)
	for i := range ds {
		ti.Put(validNames[i%len(validNames)], i)
	}

	bmarks := []struct {
		name  string
		query string
		count int
	}{
		{"Equals_____Acy", "Acy", 443},
		{"Equals____Abel", "Abel", 443},
		{"Contains_%ana%", "%ana%", 34_958},
		{"Prefix____Jo%", "Jo%", 46_072},
		{"Suffix___%son", "%son", 28_775},
		{"Both___Jo%son", "Jo%son", 443},
		{"Multi_%an%na%", "%an%na%", 26_551},
		{"All_________%", "%", ds},
	}

	var globalCount int

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				ids, _ := ti.Like(bench.query)
				count := ids.Count()

				if count != bench.count {
					b.Fatalf("%s: expected count %d, got %d", bench.name, bench.count, count)
				}

				globalCount = count
			}
		})
	}

	_ = globalCount
}
