package main

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

	list := make([]person, 0, ds)

	il := NewIndexList[person]()
	err := il.CreateIndex("name", NewSortedIndex(FromName[person, string]("Name")))
	assert.NoError(b, err)
	err = il.CreateIndex("age", NewSortedIndex(FromName[person, int]("Age")))
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

		list = append(list, person{
			Name: names[n],
			Age:  minV + rand.IntN(maxV-minV+1),
		})
	}
	fmt.Printf("- Count: %d, Time: %s\n", il.Count(), time.Since(start))
	b.ResetTimer()

	bmarks := []struct {
		name  string
		bmark func() int
	}{
		{
			name: "IndexList",
			bmark: func() int {
				qr, _ := il.QueryStr(
					`name = "Jule" or name = "Magan" or age > 80`,
				)
				return qr.Count()
			},
		},
		{
			name: "IndexList-In",
			bmark: func() int {
				qr, _ := il.QueryStr(
					`name IN("Jule", "Magan") or age > 80`,
				)
				return qr.Count()
			},
		},
		{
			name: "List",
			bmark: func() int {
				count := 0
				for _, p := range list {
					if p.Name == "Jule" || p.Name == "Magan" || p.Age > 80 {
						count++
					}
				}
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
			fmt.Printf("---%s: %d \n", bench.name, count)

		})
	}
}
