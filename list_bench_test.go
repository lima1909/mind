package mind

import (
	_ "embed"

	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//go:embed testdata/names.txt
var names_txt string

// GOGC=off go test -bench=QueryStr -cpuprofile=cpu.prof
// go tool pprof  cpu.prof
// go tool pprof -http=:8080 cpu.prof

// go test -memprofile=mem.prof
// go tool pprof -alloc_objects mem.prof  # Shows total allocations (even if GC'd)
// go tool pprof -inuse_space mem.prof    # Shows currently held memory

// go test -trace=trace.out
// go tool trace trace.out

// go test -mutexprofile=mutex.prof
// go tool pprof mutex.prof

// go test -blockprofile=block.prof
// go tool pprof block.prof

// go test  -bench=Ranges -count=2 -run=xy -benchmem

// Update: https://pkg.go.dev/github.com/lima1909/mind
// https://proxy.golang.org/github.com/lima1909/mind/@v/list

func BenchmarkQueryStr(b *testing.B) {
	type person struct {
		Name string
		Age  uint8
	}

	ds := 3_000_000
	n := 0
	names := strings.Split(names_txt, "\n")

	start := time.Now()
	fl := NewFreeListWithCapacity[person](ds)

	for i := 1; i <= ds; i++ {
		if n%6779 == 0 {
			n = 0
		}
		n++

		fl.Insert(person{
			Name: names[n],
			Age:  uint8(i % 100),
		})
	}

	il := NewList[person]()
	err := il.CreateIndex("name", NewStringIndex(FromName[person, string]("Name")))
	require.NoError(b, err)
	err = il.CreateIndex("age", NewRangeEncodedIndex(FromName[person, uint8]("Age"), 100))
	require.NoError(b, err)
	err = il.CreateIndex("age2", NewRangeIndex(FromName[person, uint8]("Age")))
	require.NoError(b, err)

	err = il.InitialBulkInsert(fl)
	require.NoError(b, err)

	fmt.Printf("- Count: %d, Time: %s\n", il.Count(), time.Since(start))

	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func() int
		count int
	}{
		{
			name: "List-Or",
			bmark: func() int {
				result := il.QueryStr(
					`name = "Jule" or name = "Magan" or age > 80`,
				)
				count, err := result.Count()
				require.NoError(b, err)
				return count
			},
			count: 570_716,
		},
		{
			name: "List-OrRg",
			bmark: func() int {
				result := il.QueryStr(
					`name = "Jule" or name = "Magan" or age2 > 80`,
				)
				count, err := result.Count()
				require.NoError(b, err)
				return count
			},
			count: 570_716,
		},
		{
			name: "List-In",
			bmark: func() int {
				result := il.QueryStr(
					`name IN("Jule", "Magan") or age > 80`,
				)
				count, err := result.Count()
				require.NoError(b, err)
				return count
			},
			count: 570_716,
		},
		{
			name: "Contains",
			bmark: func() int {
				result := il.QueryStr(
					`name like "%ule%" or name like "%agan%"`,
				)
				count, err := result.Count()
				require.NoError(b, err)
				return count
			},
			count: 3981,
		},
		{
			name: "Startswith",
			bmark: func() int {
				result := il.QueryStr(
					`name like "Jul%" or name like "Magai%"`,
				)
				count, err := result.Count()
				require.NoError(b, err)
				return count
			},
			count: 8417,
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			for b.Loop() {
				count := bench.bmark()
				if count != bench.count {
					b.Fatalf("%s: expected count %d, got %d", bench.name, bench.count, count)
				}
			}
		})
	}
}
