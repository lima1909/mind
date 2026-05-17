package mind

import (
	_ "embed"

	"fmt"
	"math/rand/v2"
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

// Update: https://pkg.go.dev/github.com/lima1909/mind
// https://proxy.golang.org/github.com/lima1909/mind/@v/list

func BenchmarkQueryStr(b *testing.B) {
	type person struct {
		Name string
		Age  uint8
	}

	minV := 10
	maxV := 100
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
			Age:  uint8(minV + rand.IntN(maxV-minV+1)),
		})
	}

	il := NewList[person]()
	err := il.CreateIndex("name", NewStringIndex(FromName[person, string]("Name")))
	require.NoError(b, err)
	err = il.CreateIndex("age", NewSortedIndex(FromName[person, uint8]("Age")))
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
	}{
		{
			name: "List-Or",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name = "Jule" or name = "Magan" or age > 80`,
				).Count()
				return count
			},
		},
		{
			name: "List-OrRg",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name = "Jule" or name = "Magan" or age2 > 80`,
				).Count()
				return count
			},
		},
		{
			name: "List-In",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name IN("Jule", "Magan") or age > 80`,
				).Count()
				return count
			},
		},
		{
			name: "Contains",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name contains "ule" or name contains "agan"`,
				).Count()
				return count
			},
		},
		{
			name: "Startswith",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name startswith "Jul" or name startswith "Maga"`,
				).Count()
				return count
			},
		},
	}

	for _, bench := range bmarks {
		b.Run(bench.name, func(b *testing.B) {
			count := 0
			for b.Loop() {
				count = max(count, bench.bmark())
			}
			// fmt.Printf("---%s: %d \n", bench.name, count)

		})
	}
}
