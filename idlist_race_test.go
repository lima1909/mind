package mind

import (
	"fmt"
	"sync"
	"testing"

	"github.com/lima1909/mind/index"
	"github.com/lima1909/mind/query"
)

// buildRaceList creates an IDList pre-populated with n cars and a couple of
// indexes, so queries have something to match against.
func buildRaceList(t *testing.T, n int) *IDList[car, string] {
	t.Helper()

	l := NewIDList((*car).Name)
	if err := l.CreateIndex("age", index.NewSortedIndex((*car).Age)); err != nil {
		t.Fatalf("create age index: %v", err)
	}
	if err := l.CreateIndex("isnew", index.NewMapIndex((*car).IsNew)); err != nil {
		t.Fatalf("create isnew index: %v", err)
	}

	for i := range n {
		l.Insert(car{
			name:  fmt.Sprintf("car-%d", i),
			age:   uint8(i % 100),
			isNew: i%2 == 0,
		})
	}

	return l
}

// TestIDList_Race_ParallelQueryUpdate runs many QHandle.Update calls in
// parallel with reading queries. All updaters match every item (query.All) and
// only touch a non-id field, so the working set stays stable while every slot
// is written concurrently.
func TestIDList_Race_ParallelQueryUpdate(t *testing.T) {
	t.Parallel()

	const (
		items      = 200
		writers    = 8
		readers    = 8
		iterations = 200
	)

	l := buildRaceList(t, items)

	var wg sync.WaitGroup

	// writers: mutate every item through the query handle
	for w := range writers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()

			for i := range iterations {
				_, _ = l.Query(query.All()).Update(func(c *car) {
					c.color = fmt.Sprintf("color-%d-%d", w, i)
				})
			}
		}(w)
	}

	// readers: read the whole working set concurrently
	for range readers {
		wg.Go(func() {
			for range iterations {
				_, _ = l.Query(query.All()).Values()
				_, _ = l.Query(query.All()).Count()
			}
		})
	}

	wg.Wait()
}

// TestIDList_Race_ParallelQueryRemove runs QHandle.Remove calls in parallel
// with a feeder that keeps re-inserting items, plus readers. The concurrent
// removes (each mutating under a read lock) race against each other, the
// feeder's Insert and the reading queries.
func TestIDList_Race_ParallelQueryRemove(t *testing.T) {
	t.Parallel()

	const (
		items      = 400
		removers   = 8
		readers    = 4
		iterations = 300
	)

	l := buildRaceList(t, items)

	// finite: removers + readers (bounded work). feeder: runs until stopped.
	var finite sync.WaitGroup
	var feeder sync.WaitGroup
	stop := make(chan struct{})

	// feeder: keep the list populated so removers always have work
	feeder.Go(func() {
		i := items
		for {
			select {
			case <-stop:
				return
			default:
				l.Insert(car{
					name:  fmt.Sprintf("car-%d", i),
					age:   uint8(i % 100),
					isNew: i%2 == 0,
				})
				i++
			}
		}
	})

	// removers: delete items matched by a query
	for rm := range removers {
		finite.Add(1)
		go func(rm int) {
			defer finite.Done()
			for i := range iterations {
				age := (rm*7 + i) % 100
				_, _ = l.QueryStr(fmt.Sprintf("age = %d", age)).Remove()
			}
		}(rm)
	}

	// readers: query concurrently with the removes
	for range readers {
		finite.Go(func() {
			for range iterations {
				_, _ = l.QueryStr("age >= 0").Count()
				_, _ = l.Query(query.All()).Values()
			}
		})
	}

	// wait for the bounded workers, then stop and join the feeder
	finite.Wait()
	close(stop)
	feeder.Wait()
}

// TestIDList_Race_MixedQHandleAndDirect mixes query-driven mutation
// (QHandle.Remove / QHandle.Update) with the directly-locked API
// (Insert / Update / Remove / Get) and reading queries, to exercise the
// interaction between the read-locked query path and the write-locked
// direct path.
func TestIDList_Race_MixedQHandleAndDirect(t *testing.T) {
	t.Parallel()

	const (
		items      = 200
		workers    = 6
		iterations = 200
	)

	l := buildRaceList(t, items)

	var wg sync.WaitGroup

	kinds := []func(id int, i int){
		// query-driven update (mutates under RLock)
		func(id, i int) {
			_, _ = l.Query(query.All()).Update(func(c *car) {
				c.color = fmt.Sprintf("c-%d-%d", id, i)
			})
		},
		// query-driven remove (mutates under RLock)
		func(id, i int) {
			_, _ = l.QueryStr(fmt.Sprintf("age = %d", i%100)).Remove()
		},
		// direct, fully-locked insert
		func(id, i int) {
			l.Insert(car{name: fmt.Sprintf("new-%d-%d", id, i), age: uint8(i % 100)})
		},
		// direct, fully-locked update by id
		func(id, i int) {
			_ = l.Update(fmt.Sprintf("car-%d", i%items), func(c *car) { c.color = "x" })
		},
		// reading query
		func(id, i int) {
			_, _ = l.Query(query.All()).Values()
		},
		// direct read
		func(id, i int) {
			_, _ = l.Get(fmt.Sprintf("car-%d", i%items))
		},
	}

	for w := range workers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()

			for i := range iterations {
				kinds[(w+i)%len(kinds)](w, i)
			}
		}(w)
	}

	wg.Wait()
}
