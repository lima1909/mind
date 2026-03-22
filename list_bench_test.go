package mind

import (
	_ "embed"

	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func BenchmarkQueryStr(b *testing.B) {
	type person struct {
		Name string
		Age  int
	}

	minV := 10
	maxV := 100
	names := strings.Split(names_txt, "\n")

	start := time.Now()

	// Sprintf is expensive, so we only test with 250_000 datasets
	ds := 3_000_000

	// list := make([]person, 0, ds)

	il := NewList[person]()
	err := il.CreateIndex("name", NewSortedIndex(FromName[person, string]("Name")))
	assert.NoError(b, err)
	err = il.CreateIndex("age", NewSortedIndex(FromName[person, int]("Age")))
	assert.NoError(b, err)

	err = il.CreateIndex("name2", NewFullScan(FromName[person, string]("Name")))
	assert.NoError(b, err)
	err = il.CreateIndex("age2", NewFullScan(FromName[person, int]("Age")))
	assert.NoError(b, err)

	n := 0
	for i := 1; i <= ds; i++ {
		if n%6779 == 0 {
			n = 0
		}
		n++

		il.Insert(person{
			Name: names[n],
			Age:  minV + rand.IntN(maxV-minV+1),
		})

		// list = append(list, person{
		// 	Name: names[n],
		// 	Age:  minV + rand.IntN(maxV-minV+1),
		// })
	}
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
			name: "List-In",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name IN("Jule", "Magan") or age > 80`,
				).Count()
				return count
			},
		},
		{
			name: "List-And",
			bmark: func() int {
				count, _ := il.QueryStr(
					`name = "Jule" or name = "Magan" and age > 80`,
				).Count()
				return count
			},
		},
		// {
		// 	name: "FullScan",
		// 	bmark: func() int {
		// 		count, _ := il.QueryStr(
		// 			`name2 IN("Jule", "Magan") or age2 > 80`,
		// 		).Count()
		// 		return count
		// 	},
		// },
		// {
		// 	name: "List-NoIdx",
		// 	bmark: func() int {
		// 		count := 0
		// 		for _, p := range list {
		// 			if p.Name == "Jule" || p.Name == "Magan" || p.Age > 80 {
		// 				count++
		// 			}
		// 			if count > 5000 {
		// 				break
		// 			}
		// 		}
		// 		return count
		// 	},
		// },
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
